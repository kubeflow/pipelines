#!/usr/bin/env node
/**
 * Test Data Seeder for UI Smoke Tests
 *
 * Creates sample pipelines, experiments, and runs to populate the UI
 * with realistic data for screenshot comparison.
 */

const http = require('http');
const https = require('https');
const fs = require('fs');
const path = require('path');

const API_BASE = process.env.API_BASE || 'http://localhost:3001';
const REPO_ROOT = path.resolve(__dirname, '../../..');
const DEFAULT_MANIFEST_PATH = path.join(REPO_ROOT, '.ui-smoke-test', 'seed-manifest.json');
const SEED_MANIFEST_PATH = process.env.UI_SMOKE_SEED_MANIFEST || DEFAULT_MANIFEST_PATH;

function log(msg, type = 'info') {
  const colors = { info: '\x1b[32m', warn: '\x1b[33m', error: '\x1b[31m', debug: '\x1b[36m' };
  const reset = '\x1b[0m';
  console.log(`${colors[type] || ''}[SEED]${reset} ${msg}`);
}

function unique(values) {
  return Array.from(new Set(values.filter(Boolean).map((v) => String(v))));
}

function pickList(response, listKeys) {
  for (const key of listKeys) {
    if (Array.isArray(response[key])) {
      return response[key];
    }
  }
  return [];
}

function extractIds(items, candidateKeys) {
  return unique(
    items.map((item) => {
      for (const key of candidateKeys) {
        if (item && item[key]) {
          return item[key];
        }
      }
      return null;
    }),
  );
}

function selectComparableRunIds(runs) {
  return extractIds(
    runs.filter((run) => {
      const pipelineSpec = run?.pipeline_spec || run?.pipelineSpec;
      if (!pipelineSpec || typeof pipelineSpec !== 'object') return false;
      const schemaVersion = pipelineSpec.schema_version || pipelineSpec.schemaVersion;
      return (
        typeof schemaVersion === 'string' &&
        schemaVersion.startsWith('2.') &&
        pipelineSpec.root !== null &&
        typeof pipelineSpec.root === 'object'
      );
    }),
    ['run_id', 'runId', 'id'],
  );
}

async function listComparableRunIds(request = apiRequest, minimumRunCount = 2) {
  let runIds = [];
  let pageToken = '';
  const seenPageTokens = new Set();
  do {
    const pageTokenQuery = pageToken ? `&page_token=${encodeURIComponent(pageToken)}` : '';
    const response = await request('GET', `/apis/v2beta1/runs?page_size=20${pageTokenQuery}`);
    runIds = unique([...runIds, ...selectComparableRunIds(pickList(response, ['runs']))]);
    const nextPageToken = response.next_page_token || response.nextPageToken || '';
    if (!nextPageToken || seenPageTokens.has(nextPageToken)) break;
    seenPageTokens.add(nextPageToken);
    pageToken = nextPageToken;
  } while (runIds.length < minimumRunCount);
  return runIds;
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function fetchResourceIds() {
  const [pipelinesResp, experimentsResp, runIds, recurringResp] = await Promise.all([
    apiRequest('GET', '/apis/v2beta1/pipelines?page_size=20'),
    apiRequest('GET', '/apis/v2beta1/experiments?page_size=20'),
    listComparableRunIds(),
    apiRequest('GET', '/apis/v2beta1/recurringruns?page_size=20'),
  ]);

  return {
    experimentIds: extractIds(pickList(experimentsResp, ['experiments']), [
      'experiment_id',
      'experimentId',
      'id',
    ]),
    pipelineIds: extractIds(pickList(pipelinesResp, ['pipelines']), [
      'pipeline_id',
      'pipelineId',
      'id',
    ]),
    recurringRunIds: extractIds(
      pickList(recurringResp, ['recurring_runs', 'recurringRuns', 'jobs']),
      ['recurring_run_id', 'recurringRunId', 'job_id', 'id'],
    ),
    runIds,
  };
}

function createdIds(created) {
  return {
    experimentIds: unique(
      (created.experiments || []).map((e) => e.experiment_id || e.experimentId || e.id),
    ),
    pipelineIds: unique(
      (created.pipelines || []).map((p) => p.pipeline_id || p.pipelineId || p.id),
    ),
    recurringRunIds: unique(
      (created.recurringRuns || []).map(
        (r) => r.recurring_run_id || r.recurringRunId || r.job_id || r.id,
      ),
    ),
    runIds: unique((created.runs || []).map((r) => r.run_id || r.runId || r.id)),
  };
}

function buildSeedManifest(resourceIds) {
  const defaults = {
    artifactId: (resourceIds.artifactIds || [])[0] || null,
    compareRunlist: resourceIds.runIds.slice(0, 3).join(','),
    experimentId: resourceIds.experimentIds[0] || null,
    pipelineId: resourceIds.pipelineIds[0] || null,
    recurringRunId: resourceIds.recurringRunIds[0] || null,
    runId: resourceIds.runIds[0] || null,
  };

  return {
    apiBase: API_BASE,
    defaults,
    generatedAt: new Date().toISOString(),
    resources: resourceIds,
  };
}

async function createRelatedTaskArtifact(runId, request = apiRequest) {
  if (!runId) {
    throw new Error('A run is required to seed the native artifact/task relationship.');
  }

  const tasksResponse = await request(
    'GET',
    `/apis/v2beta1/runs/${encodeURIComponent(runId)}/tasks?page_size=200`,
  );
  const seededTasks = pickList(tasksResponse, ['tasks']).filter(
    (task) => (task.name || '') === 'ui-smoke-related-task',
  );
  for (const seededTask of seededTasks) {
    const taskId = seededTask.task_id || seededTask.taskId || seededTask.id;
    if (!taskId) continue;
    const relationshipPath = `/apis/v2beta1/artifact_tasks?task_ids=${encodeURIComponent(taskId)}&page_size=1`;
    const relationshipResponse = await request('GET', relationshipPath);
    const relationship = pickList(relationshipResponse, ['artifact_tasks', 'artifactTasks'])[0];
    const artifactId = relationship?.artifact_id || relationship?.artifactId;
    if (artifactId) {
      log(`  ✓ Reusing related task artifact: ${artifactId}`);
      return { artifactId, taskId };
    }
  }

  log(`Creating native artifact/task relationship for run: ${runId}`);
  const task = await request('POST', `/apis/v2beta1/runs/${encodeURIComponent(runId)}/tasks`, {
    display_name: 'UI Smoke Related Task',
    name: 'ui-smoke-related-task',
    scope_path: 'root.ui-smoke-related-task',
    state: 'SUCCEEDED',
    type: 'RUNTIME',
  });
  const taskId = task.task_id || task.taskId || task.id;
  if (!taskId) {
    throw new Error('Task service returned a seeded task without an ID.');
  }

  const artifact = await request('POST', '/apis/v2beta1/artifacts', {
    artifact: {
      name: 'UI Smoke Related Tasks Artifact',
      namespace: process.env.UI_SMOKE_NAMESPACE || 'kubeflow',
      type: 'Dataset',
      uri: `minio://mlpipeline/ui-smoke/${runId}/${taskId}/artifact.json`,
    },
    producer_key: 'ui_smoke_output',
    run_id: runId,
    task_id: taskId,
  });
  const artifactId = artifact.artifact_id || artifact.artifactId || artifact.id;
  if (!artifactId) {
    throw new Error('Artifact service returned a seeded artifact without an ID.');
  }

  const relationshipPath = `/apis/v2beta1/artifact_tasks?artifact_ids=${encodeURIComponent(artifactId)}&page_size=1`;
  for (let attempt = 0; attempt < 10; attempt++) {
    const response = await request('GET', relationshipPath);
    const relationships = pickList(response, ['artifact_tasks', 'artifactTasks']);
    if (relationships.some((link) => (link.artifact_id || link.artifactId) === artifactId)) {
      log(`  ✓ Created related task artifact: ${artifactId}`);
      return { artifactId, taskId };
    }
    await delay(500);
  }
  throw new Error(
    `Artifact/task relationship for ${artifactId} was not observable after creation.`,
  );
}

function writeSeedManifest(manifest) {
  const dirPath = path.dirname(SEED_MANIFEST_PATH);
  fs.mkdirSync(dirPath, { recursive: true });
  fs.writeFileSync(SEED_MANIFEST_PATH, JSON.stringify(manifest, null, 2));
  log(`Wrote seed manifest: ${SEED_MANIFEST_PATH}`);
}

/**
 * Make an HTTP request to the KFP API
 */
async function apiRequest(method, endpoint, body = null) {
  return new Promise((resolve, reject) => {
    const url = new URL(endpoint, API_BASE);
    const options = {
      method,
      hostname: url.hostname,
      port: url.port,
      path: url.pathname + url.search,
      headers: {
        'Content-Type': 'application/json',
      },
    };

    const protocol = url.protocol === 'https:' ? https : http;
    const req = protocol.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => (data += chunk));
      res.on('end', () => {
        try {
          const json = JSON.parse(data);
          if (res.statusCode >= 400) {
            reject(new Error(`API error ${res.statusCode}: ${JSON.stringify(json)}`));
          } else {
            resolve(json);
          }
        } catch (e) {
          if (res.statusCode >= 400) {
            reject(new Error(`API error ${res.statusCode}: ${data}`));
          } else {
            resolve(data);
          }
        }
      });
    });

    req.on('error', reject);
    req.setTimeout(10000, () => {
      req.destroy();
      reject(new Error('Request timeout'));
    });

    if (body) {
      req.write(JSON.stringify(body));
    }
    req.end();
  });
}

/**
 * Check API health
 */
async function checkHealth() {
  try {
    await apiRequest('GET', '/apis/v2beta1/healthz');
    return true;
  } catch (e) {
    return false;
  }
}

/**
 * Create a sample experiment
 */
async function createExperiment(name, description) {
  log(`Creating experiment: ${name}`);
  try {
    const result = await apiRequest('POST', '/apis/v2beta1/experiments', {
      display_name: name,
      description: description,
    });
    log(`  ✓ Created experiment: ${result.experiment_id || result.name}`);
    return result;
  } catch (e) {
    log(`  ✗ Failed to create experiment: ${e.message}`, 'warn');
    return null;
  }
}

/**
 * Build a minimal pipeline spec that can be submitted directly with runs.
 */
function normalizePipelineName(name) {
  return (
    name
      .toLowerCase()
      .replace(/[^a-z0-9-]+/g, '-')
      .replace(/^-+|-+$/g, '')
      .slice(0, 63) || 'ui-smoke-pipeline'
  );
}

function buildPipelineSpec(name, description) {
  return {
    pipelineInfo: {
      name: normalizePipelineName(name),
      description: description,
    },
    deploymentSpec: {
      executors: {
        'exec-print-msg': {
          container: {
            image: 'python:3.9-slim',
            command: ['python', '-c'],
            args: ['print("Hello from ' + name + '")'],
          },
        },
      },
    },
    root: {
      dag: {
        tasks: {
          'print-msg': {
            taskInfo: { name: 'print-msg' },
            componentRef: { name: 'comp-print-msg' },
          },
        },
      },
    },
    components: {
      'comp-print-msg': {
        executorLabel: 'exec-print-msg',
      },
    },
    schemaVersion: '2.1.0',
    sdkVersion: 'kfp-2.0.0',
  };
}

/**
 * Create a sample pipeline. Runs use an embedded spec because this JSON-only smoke seeder does not
 * upload a package for the pipeline-version API.
 */
async function uploadPipeline(name, description, request = apiRequest) {
  log(`Creating pipeline: ${name}`);

  try {
    const result = await request('POST', '/apis/v2beta1/pipelines', {
      display_name: name,
      description: description,
    });
    log(`  ✓ Created pipeline: ${result.pipeline_id || result.name}`);
    return result;
  } catch (e) {
    log(`  ✗ Failed to create pipeline: ${e.message}`, 'warn');
    return null;
  }
}

/**
 * Create a pipeline run
 */
async function createRun(name, _pipelineId, experimentId, request = apiRequest) {
  log(`Creating run: ${name}`);
  try {
    const body = {
      display_name: name,
      description: `Test run for ${name}`,
      pipeline_spec: buildPipelineSpec(name, `Test run for ${name}`),
      runtime_config: {
        pipeline_root: `minio://mlpipeline/ui-smoke/${normalizePipelineName(name)}`,
      },
    };

    if (experimentId) {
      body.experiment_id = experimentId;
    }

    const result = await request('POST', '/apis/v2beta1/runs', body);
    log(`  ✓ Created run: ${result.run_id || result.name}`);
    return result;
  } catch (e) {
    log(`  ✗ Failed to create run: ${e.message}`, 'warn');
    return null;
  }
}

/**
 * Create a recurring run
 */
async function createRecurringRun(name, _pipelineId, experimentId, request = apiRequest) {
  log(`Creating recurring run: ${name}`);
  try {
    const body = {
      display_name: name,
      description: `Scheduled run for ${name}`,
      pipeline_spec: buildPipelineSpec(name, `Scheduled run for ${name}`),
      runtime_config: {
        pipeline_root: `minio://mlpipeline/ui-smoke/${normalizePipelineName(name)}`,
      },
      mode: 'ENABLE',
      max_concurrency: '1',
      trigger: {
        periodic_schedule: {
          interval_second: '3600', // Every hour
        },
      },
    };

    if (experimentId) {
      body.experiment_id = experimentId;
    }

    const result = await request('POST', '/apis/v2beta1/recurringruns', body);
    log(`  ✓ Created recurring run: ${result.recurring_run_id || result.name}`);
    return result;
  } catch (e) {
    log(`  ✗ Failed to create recurring run: ${e.message}`, 'warn');
    return null;
  }
}

/**
 * Get existing data counts
 */
async function getExistingCounts() {
  const counts = {
    pipelines: 0,
    experiments: 0,
    runs: 0,
    recurringRuns: 0,
  };

  try {
    const pipelines = await apiRequest('GET', '/apis/v2beta1/pipelines?page_size=1');
    counts.pipelines = pipelines.total_size || 0;
  } catch (e) {
    /* ignore */
  }

  try {
    const experiments = await apiRequest('GET', '/apis/v2beta1/experiments?page_size=1');
    counts.experiments = experiments.total_size || 0;
  } catch (e) {
    /* ignore */
  }

  try {
    counts.runs = (await listComparableRunIds()).length;
  } catch (e) {
    /* ignore */
  }

  try {
    const recurringRuns = await apiRequest('GET', '/apis/v2beta1/recurringruns?page_size=1');
    counts.recurringRuns = recurringRuns.total_size || 0;
  } catch (e) {
    /* ignore */
  }

  return counts;
}

function canReuseExistingData(existingCounts) {
  return (
    existingCounts.pipelines > 0 &&
    existingCounts.experiments > 0 &&
    existingCounts.runs >= 2 &&
    existingCounts.recurringRuns > 0
  );
}

/**
 * Seed test data
 */
async function seedData(options = {}) {
  const {
    pipelines: numPipelines = 3,
    experiments: numExperiments = 2,
    runs: numRuns = 5,
    recurringRuns: numRecurringRuns = 2,
    skipIfExists = true,
  } = options;

  log('Seeding test data for UI smoke tests...');
  log(`API Base: ${API_BASE}`);

  // Check API health first
  const healthy = await checkHealth();
  if (!healthy) {
    log('API is not healthy. Skipping data seeding.', 'error');
    return { success: false, error: 'API not healthy' };
  }

  // Check existing data
  const existingCounts = await getExistingCounts();
  log(`Existing data: ${JSON.stringify(existingCounts)}`);

  if (skipIfExists && canReuseExistingData(existingCounts)) {
    log('Data already exists. Skipping seeding (use --force to override).');
    try {
      const resourceIds = await fetchResourceIds();
      const relatedTaskArtifact = await createRelatedTaskArtifact(resourceIds.runIds[0]);
      resourceIds.artifactIds = [relatedTaskArtifact.artifactId];
      resourceIds.taskIds = [relatedTaskArtifact.taskId];
      writeSeedManifest(buildSeedManifest(resourceIds));
      return {
        success: true,
        skipped: true,
        counts: existingCounts,
        seedManifestPath: SEED_MANIFEST_PATH,
      };
    } catch (error) {
      log(`Failed to write seed manifest from existing data: ${error.message}`, 'warn');
      return { success: false, skipped: true, counts: existingCounts, error: error.message };
    }
  }

  const created = {
    experiments: [],
    pipelines: [],
    runs: [],
    recurringRuns: [],
  };

  // Create experiments
  const experimentNames = [
    { name: 'Image Classification', desc: 'Training and evaluating image classification models' },
    { name: 'NLP Pipeline', desc: 'Natural language processing experiments' },
    { name: 'Data Preprocessing', desc: 'Data cleaning and feature engineering' },
  ];

  for (let i = 0; i < Math.min(numExperiments, experimentNames.length); i++) {
    const exp = await createExperiment(experimentNames[i].name, experimentNames[i].desc);
    if (exp) created.experiments.push(exp);
  }

  // Create pipelines
  const pipelineNames = [
    { name: 'Training Pipeline', desc: 'End-to-end ML training pipeline' },
    { name: 'Data Ingestion', desc: 'Extract and load data from various sources' },
    { name: 'Model Evaluation', desc: 'Evaluate model performance metrics' },
    { name: 'Feature Engineering', desc: 'Transform raw data into features' },
    { name: 'Batch Inference', desc: 'Run batch predictions on new data' },
  ];

  for (let i = 0; i < Math.min(numPipelines, pipelineNames.length); i++) {
    const pipeline = await uploadPipeline(pipelineNames[i].name, pipelineNames[i].desc);
    if (pipeline) created.pipelines.push(pipeline);
  }

  // Create runs
  const runNames = [
    'Training Run - v1.0',
    'Training Run - v1.1',
    'Evaluation Run - Test Set',
    'Inference Run - Batch 1',
    'Data Processing - Jan 2024',
  ];

  const defaultExperimentId = created.experiments[0]?.experiment_id;
  const defaultPipelineId = created.pipelines[0]?.pipeline_id;

  for (let i = 0; i < Math.min(numRuns, runNames.length); i++) {
    const run = await createRun(runNames[i], defaultPipelineId, defaultExperimentId);
    if (run) created.runs.push(run);
  }

  // Create recurring runs
  const recurringNames = ['Daily Training', 'Hourly Data Sync'];

  for (let i = 0; i < Math.min(numRecurringRuns, recurringNames.length); i++) {
    const recurring = await createRecurringRun(
      recurringNames[i],
      defaultPipelineId,
      defaultExperimentId,
    );
    if (recurring) created.recurringRuns.push(recurring);
  }

  // Summary
  log('');
  log('=== Seeding Complete ===');
  log(`  Experiments: ${created.experiments.length}`);
  log(`  Pipelines: ${created.pipelines.length}`);
  log(`  Runs: ${created.runs.length}`);
  log(`  Recurring Runs: ${created.recurringRuns.length}`);

  try {
    const fetchedIds = await fetchResourceIds();
    const fromCreated = createdIds(created);
    const resourceIds = {
      artifactIds: [],
      experimentIds: unique([...fromCreated.experimentIds, ...fetchedIds.experimentIds]),
      pipelineIds: unique([...fromCreated.pipelineIds, ...fetchedIds.pipelineIds]),
      recurringRunIds: unique([...fromCreated.recurringRunIds, ...fetchedIds.recurringRunIds]),
      runIds: unique([...fromCreated.runIds, ...fetchedIds.runIds]),
      taskIds: [],
    };
    const relatedTaskArtifact = await createRelatedTaskArtifact(resourceIds.runIds[0]);
    resourceIds.artifactIds = [relatedTaskArtifact.artifactId];
    resourceIds.taskIds = [relatedTaskArtifact.taskId];
    writeSeedManifest(buildSeedManifest(resourceIds));
  } catch (error) {
    log(`Failed to generate seed manifest: ${error.message}`, 'warn');
    return { success: false, created, error: error.message };
  }

  return { success: true, created, seedManifestPath: SEED_MANIFEST_PATH };
}

/**
 * Clear all test data
 */
async function clearData() {
  log('Clearing test data...');

  // This would delete all pipelines, experiments, runs
  // For safety, we'll just log a warning
  log('Data clearing not implemented for safety. Delete cluster to reset.', 'warn');

  return { success: false, error: 'Not implemented' };
}

// CLI interface
if (require.main === module) {
  const args = process.argv.slice(2);
  const force = args.includes('--force');
  const clear = args.includes('--clear');

  if (clear) {
    clearData().then((result) => {
      process.exit(result.success ? 0 : 1);
    });
  } else {
    seedData({ skipIfExists: !force }).then((result) => {
      process.exit(result.success ? 0 : 1);
    });
  }
}

module.exports = {
  checkHealth,
  seedData,
  clearData,
  getExistingCounts,
  canReuseExistingData,
  selectComparableRunIds,
  listComparableRunIds,
  createExperiment,
  createRelatedTaskArtifact,
  uploadPipeline,
  createRun,
  createRecurringRun,
};
