#!/usr/bin/env node
/**
 * Creates deterministic API resources used by seeded UI screenshot routes.
 */

const fs = require('fs');
const http = require('http');
const https = require('https');
const path = require('path');

const {
  REVISION_FLAVORS,
  buildLogicalFixtures,
  buildSemanticDeployment,
  detectRevisionFlavor,
} = require('./semantic-manifest');
const {
  decodeGetArtifactsByIdResponse,
  encodeGetArtifactsByIdRequest,
} = require('./mlmd-protobuf');

const API_BASE = process.env.API_BASE || 'http://localhost:3001';
const REPO_ROOT = path.resolve(__dirname, '../../..');
const DEFAULT_MANIFEST_PATH = path.join(REPO_ROOT, '.ui-smoke-test', 'seed-manifest.json');
const SEED_MANIFEST_PATH = process.env.UI_SMOKE_SEED_MANIFEST || DEFAULT_MANIFEST_PATH;
const MULTIPART_BOUNDARY = '----kfp-ui-smoke-pipeline-boundary';
const SEMANTIC_MARKER = 'ui-smoke.semantic-id';
const GRPC_WEB_PROTO = 'application/grpc-web+proto';
const MLMD_ARTIFACT_METHOD = '/ml_metadata.MetadataStoreService/GetArtifactsByID';
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

const RICH_PIPELINE_YAML = `pipelineInfo:
  name: ui-smoke-rich-topology
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
      consume-metrics:
        taskInfo:
          name: consume-metrics
        cachingOptions:
          enableCache: false
        componentRef:
          name: comp-consume-metrics
        dependentTasks:
          - write-metrics
        inputs:
          artifacts:
            metrics:
              taskOutputArtifact:
                outputArtifactKey: scalar_metrics
                producerTask: write-metrics
      nested-dag:
        taskInfo:
          name: nested-dag
        componentRef:
          name: comp-nested-dag
        dependentTasks:
          - consume-metrics
          - parallel-loop
          - retry-once
      parallel-loop:
        taskInfo:
          name: parallel-loop
        componentRef:
          name: comp-parallel-loop
        parameterIterator:
          itemInput: pipelinechannel--loop-item
          items:
            raw: '["alpha", "beta"]'
      retry-once:
        taskInfo:
          name: retry-once
        cachingOptions:
          enableCache: false
        componentRef:
          name: comp-retry-once
        retryPolicy:
          backoffDuration: 0s
          backoffFactor: 2.0
          backoffMaxDuration: 3600s
          maxRetryCount: 1
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
  comp-consume-metrics:
    executorLabel: exec-consume-metrics
    inputDefinitions:
      artifacts:
        metrics:
          artifactType:
            schemaTitle: system.Metrics
            schemaVersion: 0.0.1
  comp-loop-worker:
    executorLabel: exec-loop-worker
    inputDefinitions:
      parameters:
        item:
          parameterType: STRING
  comp-nested-dag:
    dag:
      tasks:
        nested-worker:
          taskInfo:
            name: nested-worker
          cachingOptions:
            enableCache: false
          componentRef:
            name: comp-nested-worker
  comp-nested-worker:
    executorLabel: exec-nested-worker
  comp-parallel-loop:
    dag:
      tasks:
        loop-worker:
          taskInfo:
            name: loop-worker
          cachingOptions:
            enableCache: false
          componentRef:
            name: comp-loop-worker
          inputs:
            parameters:
              item:
                componentInputParameter: pipelinechannel--loop-item
    inputDefinitions:
      parameters:
        pipelinechannel--loop-item:
          parameterType: STRING
  comp-retry-once:
    executorLabel: exec-retry-once
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
    exec-consume-metrics:
      container:
        image: ${SEED_IMAGE}
        command:
          - /bin/sh
          - -ec
        args:
          - |
            test -f "$1"
            wc -c "$1"
          - ui-smoke-consume-metrics
          - "{{$.inputs.artifacts['metrics'].path}}"
    exec-loop-worker:
      container:
        image: ${SEED_IMAGE}
        command:
          - /bin/sh
          - -ec
        args:
          - |
            printf 'loop item: %s\\n' "$1"
          - ui-smoke-loop-worker
          - "{{$.inputs.parameters['item']}}"
    exec-nested-worker:
      container:
        image: ${SEED_IMAGE}
        command:
          - /bin/sh
          - -ec
        args:
          - |
            printf 'nested worker complete\\n'
          - ui-smoke-nested-worker
    exec-retry-once:
      container:
        image: ${SEED_IMAGE}
        command:
          - /bin/sh
          - -ec
        args:
          - |
            retry_index="\${KFP_RETRY_INDEX:-0}"
            if [ "$retry_index" -eq 0 ]; then
              echo 'intentional first-attempt failure' >&2
              exit 1
            fi
            echo 'retry completed'
          - ui-smoke-retry-once
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

const PIPELINE_YAML_BY_PROFILE = Object.freeze({
  metrics: MINIMAL_PIPELINE_YAML,
  'rich-topology': RICH_PIPELINE_YAML,
});

const RESOURCE_DEFINITIONS = {
  experiments: [
    {
      semanticKey: 'experiment.image-classification',
      displayName: 'UI Smoke - Image Classification',
      description: 'Deterministic UI smoke-test experiment',
    },
    {
      semanticKey: 'experiment.natural-language-processing',
      displayName: 'UI Smoke - Natural Language Processing',
      description: 'Second deterministic UI smoke-test experiment',
    },
  ],
  pipelines: [
    {
      semanticKey: 'pipeline.training',
      name: 'ui-smoke-training-pipeline',
      displayName: 'UI Smoke Training Pipeline',
      description: 'Deterministic training pipeline for UI screenshots',
      fixtureProfile: 'rich-topology',
    },
    {
      semanticKey: 'pipeline.data-ingestion',
      name: 'ui-smoke-data-ingestion',
      displayName: 'UI Smoke Data Ingestion',
      description: 'Deterministic data-ingestion pipeline for UI screenshots',
      fixtureProfile: 'metrics',
    },
    {
      semanticKey: 'pipeline.model-evaluation',
      name: 'ui-smoke-model-evaluation',
      displayName: 'UI Smoke Model Evaluation',
      description: 'Deterministic evaluation pipeline for UI screenshots',
      fixtureProfile: 'metrics',
    },
  ],
  runs: [
    {
      semanticKey: 'run.training-1',
      displayName: 'UI Smoke Training Run 1',
      fixtureProfile: 'rich-topology',
      pipelineSemanticKey: 'pipeline.training',
    },
    {
      semanticKey: 'run.training-2',
      displayName: 'UI Smoke Training Run 2',
      fixtureProfile: 'metrics',
      pipelineSemanticKey: 'pipeline.data-ingestion',
    },
    {
      semanticKey: 'run.evaluation',
      displayName: 'UI Smoke Evaluation Run',
      fixtureProfile: 'metrics',
      pipelineSemanticKey: 'pipeline.model-evaluation',
    },
    {
      semanticKey: 'run.inference',
      displayName: 'UI Smoke Inference Run',
      fixtureProfile: 'metrics',
      pipelineSemanticKey: 'pipeline.data-ingestion',
    },
    {
      semanticKey: 'run.data-processing',
      displayName: 'UI Smoke Data Processing Run',
      fixtureProfile: 'metrics',
      pipelineSemanticKey: 'pipeline.model-evaluation',
    },
  ],
  recurringRuns: [
    { semanticKey: 'recurring-run.daily-training', displayName: 'UI Smoke Daily Training' },
    { semanticKey: 'recurring-run.hourly-data-sync', displayName: 'UI Smoke Hourly Data Sync' },
  ],
};

function semanticDescription(description, semanticKey) {
  if (!semanticKey) return description;
  return `${description} [${SEMANTIC_MARKER}=${semanticKey}]`;
}

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
  const {
    apiBase = API_BASE,
    headers = {},
    rawBody = null,
    responseType = 'json',
    timeout = 10000,
  } = options;
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
        const chunks = [];
        if (responseType !== 'buffer') response.setEncoding('utf8');
        response.on('data', (chunk) => {
          chunks.push(chunk);
        });
        response.on('end', () => {
          const data =
            responseType === 'buffer'
              ? Buffer.concat(
                  chunks.map((chunk) => (Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk))),
                )
              : chunks.join('');
          let parsed = data;
          if (responseType !== 'buffer' && data) {
            try {
              parsed = JSON.parse(data);
            } catch (error) {
              // Preserve non-JSON bodies for actionable error messages.
            }
          } else if (responseType !== 'buffer') {
            parsed = {};
          }
          if (response.statusCode < 200 || response.statusCode >= 300) {
            const errorBody = Buffer.isBuffer(parsed)
              ? parsed.toString('utf8')
              : JSON.stringify(parsed);
            reject(new Error(`API error ${response.statusCode}: ${errorBody}`));
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

function encodeGrpcWebRequest(serializedMessage) {
  const message = Buffer.from(serializedMessage);
  const frame = Buffer.alloc(5 + message.length);
  frame[0] = 0x00;
  frame.writeUInt32BE(message.length, 1);
  message.copy(frame, 5);
  return frame;
}

function decodeGrpcWebResponse(responseBody) {
  const buffer = Buffer.from(responseBody);
  const dataFrames = [];
  let offset = 0;
  while (offset + 5 <= buffer.length) {
    const frameType = buffer[offset];
    const frameLength = buffer.readUInt32BE(offset + 1);
    const frameEnd = offset + 5 + frameLength;
    if (frameEnd > buffer.length) {
      throw new Error(
        `MLMD gRPC-web frame at offset ${offset} claims ${frameLength} bytes, but only ${buffer.length - offset - 5} remain.`,
      );
    }

    const payload = buffer.subarray(offset + 5, frameEnd);
    if (frameType === 0x00) {
      dataFrames.push(payload);
    } else if (frameType === 0x80) {
      const trailers = payload.toString('utf8');
      const status = trailers.match(/grpc-status:\s*(\d+)/i);
      if (!status || status[1] !== '0') {
        const message = trailers.match(/grpc-message:\s*([^\r\n]+)/i)?.[1] || 'unknown';
        throw new Error(
          `MLMD gRPC-web request failed with status ${status?.[1] || 'unknown'}: ${message}`,
        );
      }
    } else {
      throw new Error(`Unsupported MLMD gRPC-web frame type 0x${frameType.toString(16)}.`);
    }
    offset = frameEnd;
  }

  if (offset !== buffer.length) {
    throw new Error(`MLMD gRPC-web response contained ${buffer.length - offset} trailing byte(s).`);
  }
  if (dataFrames.length !== 1) {
    throw new Error(
      `MLMD gRPC-web response contained ${dataFrames.length} data frame(s), expected 1.`,
    );
  }
  return dataFrames[0];
}

async function fetchMlmdArtifactsByIds(artifactIds, request = apiRequest, options = {}) {
  const ids = unique(artifactIds);
  if (ids.length === 0) return [];
  const numericIds = ids.map((id) => Number(id));
  if (numericIds.some((id) => !Number.isSafeInteger(id) || id <= 0)) {
    throw new Error(`MLMD artifact IDs must be positive safe integers: ${ids.join(', ')}`);
  }

  const responseBody = await request('POST', MLMD_ARTIFACT_METHOD, null, {
    apiBase: options.apiBase || API_BASE,
    headers: {
      Accept: GRPC_WEB_PROTO,
      'Content-Type': GRPC_WEB_PROTO,
      'x-grpc-web': '1',
    },
    rawBody: encodeGrpcWebRequest(encodeGetArtifactsByIdRequest(numericIds)),
    responseType: 'buffer',
  });
  return decodeGetArtifactsByIdResponse(decodeGrpcWebResponse(responseBody));
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
  const { apiBase = API_BASE, semantic } = options;
  const manifest = {
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
  if (semantic) manifest.semantic = semantic;
  return manifest;
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
  const { displayName = name, pipelineYaml = MINIMAL_PIPELINE_YAML } = options;
  const query = new URLSearchParams({
    name,
    display_name: displayName,
    description: semanticDescription(description, options.semanticKey),
  });
  const multipart = createMultipartUpload(pipelineYaml);
  const result = await request('POST', `/apis/v2beta1/pipelines/upload?${query}`, null, {
    headers: multipart.headers,
    rawBody: multipart.body,
  });
  requireResourceId(result, ['pipeline_id', 'pipelineId', 'id'], `Pipeline ${name}`);
  return result;
}

async function uploadPipelineVersion(pipelineId, request = apiRequest, options = {}) {
  if (!pipelineId) throw new Error('A pipeline ID is required to upload a pipeline version.');
  const semanticKey = options.semanticKey ? `${options.semanticKey}.version` : null;
  const query = new URLSearchParams({
    name: 'ui-smoke-version',
    display_name: 'UI Smoke Version',
    pipelineid: pipelineId,
    description: semanticDescription(
      'Deterministic pipeline version for UI screenshots',
      semanticKey,
    ),
  });
  const multipart = createMultipartUpload(options.pipelineYaml || MINIMAL_PIPELINE_YAML);
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
  options = {},
) {
  if (!pipelineId || !experimentId) {
    throw new Error(`Run ${name} requires both a pipeline ID and an experiment ID.`);
  }
  const reference = { pipeline_id: pipelineId };
  if (pipelineVersionId) reference.pipeline_version_id = pipelineVersionId;
  const result = await request('POST', '/apis/v2beta1/runs', {
    display_name: name,
    description: semanticDescription(
      `Deterministic UI smoke-test run: ${name}`,
      options.semanticKey,
    ),
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
  options = {},
) {
  if (!pipelineId || !experimentId) {
    throw new Error(`Recurring run ${name} requires both a pipeline ID and an experiment ID.`);
  }
  const reference = { pipeline_id: pipelineId };
  if (pipelineVersionId) reference.pipeline_version_id = pipelineVersionId;
  const result = await request('POST', '/apis/v2beta1/recurringruns', {
    display_name: name,
    description: semanticDescription(
      `Deterministic UI smoke-test schedule: ${name}`,
      options.semanticKey,
    ),
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

function pipelineYamlForDefinition(definition) {
  const profile = definition?.fixtureProfile || 'metrics';
  const pipelineYaml = PIPELINE_YAML_BY_PROFILE[profile];
  if (!pipelineYaml) throw new Error(`Unknown fixture profile: ${profile}`);
  return pipelineYaml;
}

function selectedPipelineReference(selections, semanticKey) {
  const pipeline = selections.pipelines[semanticKey]?.resource;
  const version = selections.pipelineVersions[semanticKey]?.resource;
  const pipelineId = resourceId(pipeline, ['pipeline_id', 'pipelineId', 'id']);
  const pipelineVersionId = resourceId(version, ['pipeline_version_id', 'pipelineVersionId', 'id']);
  return pipelineId ? { pipelineId, pipelineVersionId } : null;
}

function createSemanticSelections() {
  return {
    experiments: {},
    pipelines: {},
    pipelineVersions: {},
    recurringRuns: {},
    runs: {},
  };
}

function selectSemanticResource(selections, kind, definition, resource) {
  if (!resource) return;
  selections[kind][definition.semanticKey] = { definition, resource };
}

function buildSemanticResourceBindings(selections) {
  const configurations = {
    experiments: {
      idKeys: ['experiment_id', 'experimentId', 'id'],
      kind: 'experiment',
    },
    pipelines: {
      idKeys: ['pipeline_id', 'pipelineId', 'id'],
      kind: 'pipeline',
    },
    pipelineVersions: {
      idKeys: ['pipeline_version_id', 'pipelineVersionId', 'id'],
      kind: 'pipeline-version',
      suffix: '.version',
    },
    recurringRuns: {
      idKeys: ['recurring_run_id', 'recurringRunId', 'job_id', 'id'],
      kind: 'recurring-run',
    },
    runs: {
      idKeys: ['run_id', 'runId', 'id'],
      kind: 'run',
    },
  };
  const bindings = {};
  for (const [selectionKind, entries] of Object.entries(selections)) {
    const configuration = configurations[selectionKind];
    for (const [definitionKey, selection] of Object.entries(entries)) {
      const semanticKey = `${definitionKey}${configuration.suffix || ''}`;
      bindings[semanticKey] = {
        displayName: selection.definition.displayName || selection.definition.name,
        id: resourceId(selection.resource, configuration.idKeys),
        kind: configuration.kind,
      };
      if (selection.definition.fixtureProfile) {
        bindings[semanticKey].fixtureProfile = selection.definition.fixtureProfile;
      }
      if (selection.definition.pipelineSemanticKey) {
        bindings[semanticKey].pipeline = selection.definition.pipelineSemanticKey;
      }
      if (configuration.kind === 'pipeline-version') {
        bindings[semanticKey].pipeline = definitionKey;
      }
    }
  }
  return bindings;
}

function legacyArtifactIdsForRun(response) {
  const run = response?.run?.run || response?.run || response || {};
  const details = run.run_details || run.runDetails || {};
  const tasks = details.task_details || details.taskDetails || [];
  const artifactIds = [];
  for (const task of tasks) {
    for (const direction of ['inputs', 'outputs']) {
      for (const artifactList of Object.values(task?.[direction] || {})) {
        artifactIds.push(...(artifactList?.artifact_ids || artifactList?.artifactIds || []));
      }
    }
  }
  return unique(artifactIds).sort((left, right) =>
    left.localeCompare(right, 'en', { numeric: true }),
  );
}

async function hydrateLegacyRunArtifacts(response, request, options = {}) {
  const artifactIds = legacyArtifactIdsForRun(response);
  if (artifactIds.length === 0) return response;
  const fetchLegacyArtifacts =
    options.fetchLegacyArtifacts ||
    ((ids) => fetchMlmdArtifactsByIds(ids, request, { apiBase: options.apiBase }));
  const semanticArtifacts = await fetchLegacyArtifacts(artifactIds);
  if (!Array.isArray(semanticArtifacts)) {
    throw new Error('Legacy MLMD artifact discovery did not return an artifact array.');
  }
  return { ...response, semanticArtifacts };
}

async function fetchRunBindingResponse(runId, request = apiRequest, options = {}) {
  const endpoint = `/apis/v2beta1/runs/${encodeURIComponent(runId)}`;
  let fullResponse;
  let fullError;
  try {
    fullResponse = await request('GET', `${endpoint}?view=FULL`);
  } catch (error) {
    fullError = error;
  }
  // A legacy FULL response owns its task/artifact projection, but its artifact lists contain
  // only MLMD IDs. Hydrate the actual MLMD values before semantic validation.
  if (detectRevisionFlavor(fullResponse) === REVISION_FLAVORS.LEGACY) {
    return hydrateLegacyRunArtifacts(fullResponse, request, options);
  }
  // Native run responses expose tasks through a revision-specific paginated endpoint, even when
  // task_count makes the run response itself look native.
  const embeddedTasks =
    fullResponse?.tasks || fullResponse?.run?.tasks || fullResponse?.run?.run?.tasks;
  if (Array.isArray(embeddedTasks) && embeddedTasks.length > 0) return fullResponse;

  let detailResponse;
  let detailError;
  try {
    detailResponse = fullResponse || (await request('GET', endpoint));
  } catch (error) {
    detailError = error;
  }
  if (detectRevisionFlavor(detailResponse) === REVISION_FLAVORS.LEGACY) {
    return hydrateLegacyRunArtifacts(detailResponse, request, options);
  }

  const runResponse = detailResponse || fullResponse;
  if (!runResponse) {
    const fullMessage = fullError ? `; FULL request failed: ${fullError.message}` : '';
    throw new Error(`Run detail request failed: ${detailError.message}${fullMessage}`);
  }

  try {
    const tasks = await listAll(`${endpoint}/tasks`, ['tasks'], request);
    if (runResponse?.run?.run && typeof runResponse.run.run === 'object') {
      return {
        ...runResponse,
        run: { ...runResponse.run, run: { ...runResponse.run.run, tasks } },
      };
    }
    if (runResponse?.run && typeof runResponse.run === 'object') {
      return { ...runResponse, run: { ...runResponse.run, tasks } };
    }
    return { ...runResponse, tasks };
  } catch (taskError) {
    throw new Error(`Native task API request failed for run ${runId}: ${taskError.message}`);
  }
}

async function fetchRunBindingResponses(selections, request = apiRequest, options = {}) {
  return Promise.all(
    Object.entries(selections.runs).map(async ([semanticKey, selection]) => ({
      response: await fetchRunBindingResponse(
        requireResourceId(
          selection.resource,
          ['run_id', 'runId', 'id'],
          `Semantic run ${semanticKey}`,
        ),
        request,
        options,
      ),
      semanticKey,
    })),
  );
}

async function waitForSemanticBindings(selections, request = apiRequest, options = {}) {
  const {
    interval = 1000,
    now = Date.now,
    sleep = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds)),
    timeout = 60000,
  } = options;
  const deadline = now() + timeout;
  const logical = buildLogicalFixtures(RESOURCE_DEFINITIONS);
  const resourceBindings = buildSemanticResourceBindings(selections);
  let lastApiError = null;
  let lastSemantic = null;

  do {
    try {
      lastSemantic = buildSemanticDeployment({
        logical,
        resourceBindings,
        runResponses: await fetchRunBindingResponses(selections, request, {
          apiBase: options.apiBase,
          fetchLegacyArtifacts: options.fetchLegacyArtifacts,
        }),
      });
      lastApiError = null;
      if (lastSemantic.validation.valid) return lastSemantic;
    } catch (error) {
      lastApiError = error;
    }

    if (now() >= deadline) break;
    await sleep(interval);
  } while (now() <= deadline);

  const error = new Error(
    lastSemantic
      ? `Timed out waiting for semantic fixtures: ${lastSemantic.validation.errors.join('; ')}`
      : `Timed out querying semantic fixture bindings: ${lastApiError?.message || 'no run details were returned'}`,
  );
  error.code = lastSemantic ? 'MISSING_FIXTURE' : 'API_INCOMPATIBILITY';
  throw error;
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
  const semanticSelections = createSemanticSelections();
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
        resource = await createExperiment(
          definition.displayName,
          semanticDescription(definition.description, definition.semanticKey),
          request,
        );
        created.experiments.push(resource);
      } catch (error) {
        failures.push(failureRecord('experiment', definition.displayName, error));
      }
    }
    if (resource) {
      selected.experiments.push(resource);
      selectSemanticResource(semanticSelections, 'experiments', definition, resource);
    }
  }

  for (const definition of RESOURCE_DEFINITIONS.pipelines.slice(0, targets.pipelines)) {
    let resource = byPipelineDefinition(inventory.pipelines, definition);
    if (!resource) {
      try {
        resource = await uploadPipeline(definition.name, definition.description, request, {
          displayName: definition.displayName,
          pipelineYaml: pipelineYamlForDefinition(definition),
          semanticKey: definition.semanticKey,
        });
        created.pipelines.push(resource);
      } catch (error) {
        failures.push(failureRecord('pipeline', definition.name, error));
      }
    }
    if (resource) {
      selected.pipelines.push(resource);
      selectSemanticResource(semanticSelections, 'pipelines', definition, resource);
    }
  }

  for (const definition of RESOURCE_DEFINITIONS.pipelines.slice(0, targets.pipelines)) {
    const pipeline = semanticSelections.pipelines[definition.semanticKey]?.resource;
    if (!pipeline) continue;
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
        const version = await uploadPipelineVersion(pipelineId, request, {
          pipelineYaml: pipelineYamlForDefinition(definition),
          semanticKey: definition.semanticKey,
        });
        created.pipelineVersions.push(version);
        versions = [version];
      }
      requireResourceId(
        versions[0],
        ['pipeline_version_id', 'pipelineVersionId', 'id'],
        `Pipeline version for ${pipelineId}`,
      );
      selected.pipelineVersions.push(versions[0]);
      selectSemanticResource(semanticSelections, 'pipelineVersions', definition, versions[0]);
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

  for (const definition of RESOURCE_DEFINITIONS.runs.slice(0, targets.runs)) {
    let resource = byDisplayName(inventory.runs, definition.displayName);
    if (!resource) {
      try {
        const pipelineReference = selectedPipelineReference(
          semanticSelections,
          definition.pipelineSemanticKey,
        ) || {
          pipelineId: primaryPipelineId,
          pipelineVersionId: primaryPipelineVersionId,
        };
        resource = await createRun(
          definition.displayName,
          pipelineReference.pipelineId,
          primaryExperimentId,
          request,
          pipelineReference.pipelineVersionId,
          { semanticKey: definition.semanticKey },
        );
        created.runs.push(resource);
      } catch (error) {
        failures.push(failureRecord('run', definition.displayName, error));
      }
    }
    if (resource) {
      selected.runs.push(resource);
      selectSemanticResource(semanticSelections, 'runs', definition, resource);
    }
  }

  for (const definition of RESOURCE_DEFINITIONS.recurringRuns.slice(0, targets.recurringRuns)) {
    let resource = byDisplayName(inventory.recurringRuns, definition.displayName);
    if (!resource) {
      try {
        resource = await createRecurringRun(
          definition.displayName,
          primaryPipelineId,
          primaryExperimentId,
          request,
          primaryPipelineVersionId,
          { semanticKey: definition.semanticKey },
        );
        created.recurringRuns.push(resource);
      } catch (error) {
        failures.push(failureRecord('recurring-run', definition.displayName, error));
      }
    }
    if (resource) {
      selected.recurringRuns.push(resource);
      selectSemanticResource(semanticSelections, 'recurringRuns', definition, resource);
    }
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

  let semantic;
  try {
    const waitForSemanticBindingsFn = options.waitForSemanticBindingsFn || waitForSemanticBindings;
    semantic = await waitForSemanticBindingsFn(semanticSelections, request, {
      apiBase,
      fetchLegacyArtifacts: options.fetchLegacyArtifacts,
      interval: options.semanticPollInterval,
      now: options.semanticNow,
      sleep: options.semanticSleep,
      timeout: options.semanticTimeout,
    });
  } catch (error) {
    return {
      success: false,
      error: `Semantic binding discovery failed: ${error.message}`,
      failureType: error.code || 'SEMANTIC_DISCOVERY_FAILURE',
      failures,
      created,
      resources,
    };
  }

  const manifest = buildSeedManifest(resources, { apiBase, semantic });
  writeSeedManifest(manifest, manifestPath);
  return {
    success: true,
    skipped: Object.values(created).every((items) => items.length === 0),
    created,
    resources,
    semantic,
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
  PIPELINE_YAML_BY_PROFILE,
  RICH_PIPELINE_YAML,
  RESOURCE_DEFINITIONS,
  SEED_IMAGE,
  SEED_MANIFEST_PATH,
  SEMANTIC_MARKER,
  apiRequest,
  buildSemanticResourceBindings,
  buildSeedManifest,
  checkHealth,
  clearData,
  createExperiment,
  createMultipartUpload,
  createRecurringRun,
  createRun,
  fetchInventory,
  fetchMlmdArtifactsByIds,
  fetchRunBindingResponse,
  fetchRunBindingResponses,
  fetchResourceIds,
  getExistingCounts,
  listAll,
  pipelineYamlForDefinition,
  resourceId,
  resolveApiUrl,
  semanticDescription,
  seedData,
  uploadPipeline,
  uploadPipelineVersion,
  validateDetailRoutes,
  waitForSemanticBindings,
  waitForRunsStable,
  writeSeedManifest,
};
