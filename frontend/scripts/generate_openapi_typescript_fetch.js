#!/usr/bin/env node
'use strict';

const fs = require('fs');
const os = require('os');
const path = require('path');
const { spawn, spawnSync } = require('child_process');

// Keep this pinned to the latest stable non-snapshot release.
const GENERATOR_IMAGE =
  process.env.OPENAPI_GENERATOR_IMAGE || 'openapitools/openapi-generator-cli:v7.19.0';
const BASE_ADDITIONAL_PROPERTIES = [
  'useSingleRequestParameter=false',
  'paramNaming=original',
  'modelPropertyNaming=original',
  'enumPropertyNaming=original',
];
const GLOBAL_PROPERTIES = [
  'apiDocs=false',
  'modelDocs=false',
  'apiTests=false',
  'modelTests=false',
].join(',');

const DEFAULT_CONCURRENCY = Math.max(1, Math.min(os.cpus().length, 4));

const SPEC_TARGETS = {
  'v1:experiment': {
    spec: 'backend/api/v1beta1/swagger/experiment.swagger.json',
    output: 'frontend/src/apis/experiment',
  },
  'v1:job': {
    spec: 'backend/api/v1beta1/swagger/job.swagger.json',
    output: 'frontend/src/apis/job',
  },
  'v1:pipeline': {
    spec: 'backend/api/v1beta1/swagger/pipeline.swagger.json',
    output: 'frontend/src/apis/pipeline',
  },
  'v1:run': {
    spec: 'backend/api/v1beta1/swagger/run.swagger.json',
    output: 'frontend/src/apis/run',
  },
  'v1:filter': {
    spec: 'backend/api/v1beta1/swagger/filter.swagger.json',
    output: 'frontend/src/apis/filter',
  },
  'v1:visualization': {
    spec: 'backend/api/v1beta1/swagger/visualization.swagger.json',
    output: 'frontend/src/apis/visualization',
  },
  'v1:auth': {
    spec: 'backend/api/v1beta1/swagger/auth.swagger.json',
    output: 'frontend/server/src/generated/apis/auth',
  },
  'v2beta1:experiment': {
    spec: 'backend/api/v2beta1/swagger/experiment.swagger.json',
    output: 'frontend/src/apisv2beta1/experiment',
  },
  'v2beta1:recurringrun': {
    spec: 'backend/api/v2beta1/swagger/recurring_run.swagger.json',
    output: 'frontend/src/apisv2beta1/recurringrun',
  },
  'v2beta1:pipeline': {
    spec: 'backend/api/v2beta1/swagger/pipeline.swagger.json',
    output: 'frontend/src/apisv2beta1/pipeline',
  },
  'v2beta1:run': {
    spec: 'backend/api/v2beta1/swagger/run.swagger.json',
    output: 'frontend/src/apisv2beta1/run',
  },
  'v2beta1:filter': {
    spec: 'backend/api/v2beta1/swagger/filter.swagger.json',
    output: 'frontend/src/apisv2beta1/filter',
  },
  'v2beta1:visualization': {
    spec: 'backend/api/v2beta1/swagger/visualization.swagger.json',
    output: 'frontend/src/apisv2beta1/visualization',
  },
  'v2beta1:auth': {
    spec: 'backend/api/v2beta1/swagger/auth.swagger.json',
    output: 'frontend/server/src/generated/apisv2beta1/auth',
  },
};

const GROUPS = {
  v1: Object.keys(SPEC_TARGETS).filter((key) => key.startsWith('v1:')),
  v2beta1: Object.keys(SPEC_TARGETS).filter((key) => key.startsWith('v2beta1:')),
  all: Object.keys(SPEC_TARGETS),
};
const SHARED_OPENAPI_SUPPORT_GROUPS = [
  {
    outputPrefixes: ['frontend/src/apis/', 'frontend/src/apisv2beta1/'],
    sharedRoot: 'frontend/src/generated/openapi',
    importExtension: '',
  },
  {
    outputPrefixes: ['frontend/server/src/generated/apis/', 'frontend/server/src/generated/apisv2beta1/'],
    sharedRoot: 'frontend/server/src/generated/openapi',
    importExtension: '.js',
  },
];
const SHARED_OPENAPI_SUPPORT_FILES = [
  { relativePath: 'runtime.ts' },
  { relativePath: path.join('models', 'ProtobufAny.ts') },
  { relativePath: path.join('models', 'GooglerpcStatus.ts') },
];

function fatal(message) {
  console.error(message);
  process.exit(1);
}

function ensureTrailingNewline(source) {
  return source.endsWith('\n') ? source : `${source}\n`;
}

function writeFileIfChanged(filePath, source) {
  const nextSource = ensureTrailingNewline(source);
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  if (fs.existsSync(filePath) && fs.readFileSync(filePath, 'utf8') === nextSource) {
    return;
  }
  fs.writeFileSync(filePath, nextSource);
}

function toImportSpecifier(filePath) {
  return filePath.split(path.sep).join('/').replace(/\.ts$/, '');
}

function resolveTargets(args) {
  if (args.length === 0) {
    return GROUPS.all;
  }

  const resolved = [];
  for (const arg of args) {
    if (GROUPS[arg]) {
      resolved.push(...GROUPS[arg]);
      continue;
    }

    if (SPEC_TARGETS[arg]) {
      resolved.push(arg);
      continue;
    }

    if (SPEC_TARGETS[`v1:${arg}`]) {
      resolved.push(`v1:${arg}`);
      continue;
    }
    if (SPEC_TARGETS[`v2beta1:${arg}`]) {
      resolved.push(`v2beta1:${arg}`);
      continue;
    }

    fatal(
      `Unknown target "${arg}". Valid groups: ${Object.keys(GROUPS).join(', ')}. Valid targets: ${Object.keys(
        SPEC_TARGETS,
      ).join(', ')}`,
    );
  }

  return [...new Set(resolved)];
}

function getSharedOpenApiSupportGroup(target) {
  const normalizedOutput = target.output.split(path.sep).join('/');
  return SHARED_OPENAPI_SUPPORT_GROUPS.find((group) =>
    group.outputPrefixes.some((prefix) => normalizedOutput.startsWith(prefix)),
  );
}

function runCommand(command, args) {
  const result = spawnSync(command, args, { stdio: 'inherit' });
  if (result.error) {
    fatal(`Failed to run ${command}: ${result.error.message}`);
  }
  if (result.status !== 0) {
    process.exit(result.status || 1);
  }
}

const MAX_STDERR_BYTES = 1024 * 1024;

function runCommandAsync(command, args) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, { stdio: ['ignore', 'ignore', 'pipe'] });
    const stderrChunks = [];
    let stderrLength = 0;
    child.stderr.on('data', (chunk) => {
      if (stderrLength >= MAX_STDERR_BYTES) return;
      const remaining = MAX_STDERR_BYTES - stderrLength;
      const toStore = chunk.length <= remaining ? chunk : chunk.subarray(0, remaining);
      stderrChunks.push(toStore);
      stderrLength += toStore.length;
    });
    child.on('error', (err) => reject(new Error(`Failed to run ${command}: ${err.message}`)));
    child.on('close', (code) => {
      if (code !== 0) {
        const error = new Error(`${command} exited with code ${code}`);
        error.stderr = Buffer.concat(stderrChunks).toString();
        reject(error);
      } else {
        resolve();
      }
    });
  });
}

function ensureDockerAvailable() {
  const result = spawnSync('docker', ['--version'], { encoding: 'utf8' });
  if (result.error || result.status !== 0) {
    fatal('Docker is required for API generation. Install/start Docker and retry `npm run apis`.');
  }
}

function ensureImageAvailable() {
  const result = spawnSync('docker', ['image', 'inspect', GENERATOR_IMAGE], {
    stdio: 'ignore',
  });
  if (result.status !== 0) {
    console.log(`Pulling ${GENERATOR_IMAGE}...`);
    runCommand('docker', ['pull', GENERATOR_IMAGE]);
  }
}

function removeGeneratorMetadata(outputDir) {
  fs.rmSync(path.join(outputDir, '.openapi-generator'), {
    recursive: true,
    force: true,
  });
  fs.rmSync(path.join(outputDir, '.openapi-generator-ignore'), {
    force: true,
  });
}

function normalizeV1ApiSymbols(source) {
  let updated = source;
  updated = updated.replace(/\b([A-Z][A-Za-z0-9_]*)V1Request\b/g, '$1Request');
  updated = updated.replace(/\b([a-z][A-Za-z0-9_]*)V1Raw\b/g, '$1Raw');
  updated = updated.replace(/\b([a-z][A-Za-z0-9_]*)V1(?=\s*\()/g, '$1');
  return updated;
}

function normalizeNodeCompatibleFetchTypes(source) {
  let updated = source;
  updated = updated.replace(/WindowOrWorkerGlobalScope\['fetch'\]/g, 'typeof fetch');
  updated = updated.replace(/\bRequestCredentials\b/g, "RequestInit['credentials']");
  // Keep the generated runtime compatible with Node-only type libs.
  updated = updated.replace(
    /export type FetchAPI = typeof fetch;/g,
    'export type FetchAPI = (input: string, init: RequestInit) => Promise<Response>;',
  );
  updated = updated.replace(
    /private fetchApi = async \(url: string, init: RequestInit\) => {/g,
    'private fetchApi = async (url: string, init: RequestInit): Promise<Response> => {',
  );
  return updated;
}

function normalizeGeneratedTypeScriptSource(source, options = {}) {
  const { nodeCompatibleFetchTypes = false, renameV1ApiSymbols = false } = options;
  let updated = source;
  if (renameV1ApiSymbols) {
    updated = normalizeV1ApiSymbols(updated);
  }
  if (nodeCompatibleFetchTypes) {
    updated = normalizeNodeCompatibleFetchTypes(updated);
  }
  return updated;
}

function normalizeGeneratedTypeScript(outputDir, options = {}) {
  if (!fs.existsSync(outputDir)) {
    return;
  }

  const { nodeCompatibleFetchTypes = false, renameV1ApiSymbols = false } = options;
  const tsFiles = listTypeScriptFiles(outputDir);
  for (const filePath of tsFiles) {
    const relativePath = path.relative(outputDir, filePath);
    const isApiFile = relativePath.split(path.sep)[0] === 'apis';
    const original = fs.readFileSync(filePath, 'utf8');
    const updated = normalizeGeneratedTypeScriptSource(original, {
      nodeCompatibleFetchTypes,
      renameV1ApiSymbols: renameV1ApiSymbols && isApiFile,
    });

    if (updated !== original) {
      fs.writeFileSync(filePath, updated);
    }
  }
}

function stripGeneratedJSDocComments(source) {
  return source.replace(/\/\*\*[\s\S]*?\*\/\n*/g, '');
}

function createSharedOpenApiSupportSource(source) {
  const lines = source.split('\n');
  const preservedPrefix = [];

  while (
    lines.length > 0 &&
    (lines[0] === '/* tslint:disable */' || lines[0] === '/* eslint-disable */')
  ) {
    preservedPrefix.push(lines.shift());
  }

  const strippedBody = stripGeneratedJSDocComments(lines.join('\n'))
    .replace(/^\s+/, '')
    .replace(/\n{3,}/g, '\n\n')
    .trimStart();

  return [
    ...preservedPrefix,
    '// Shared OpenAPI support generated by `npm run apis:all`. Do not edit manually.',
    strippedBody,
  ]
    .filter(Boolean)
    .join('\n');
}

function createOpenApiReExportShim(sharedFilePath, sourceFilePath, importExtension = '') {
  const relativeImportPath = toImportSpecifier(path.relative(path.dirname(sourceFilePath), sharedFilePath));
  const importSpecifier = `${relativeImportPath}${importExtension}`;
  return [
    '/* tslint:disable */',
    '/* eslint-disable */',
    '// Shared OpenAPI support generated by `npm run apis:all`. Do not edit manually.',
    `export * from '${importSpecifier}';`,
  ].join('\n');
}

function assertSharedOpenApiSupportMatches(sharedFilePath, sharedSource, sourceFilePath) {
  if (!fs.existsSync(sharedFilePath)) {
    return;
  }

  const expectedSharedSource = ensureTrailingNewline(sharedSource);
  const existingSharedSource = fs.readFileSync(sharedFilePath, 'utf8');
  if (existingSharedSource === expectedSharedSource) {
    return;
  }

  throw new Error(
    [
      `Shared OpenAPI support mismatch for ${path.basename(sharedFilePath)}.`,
      `Existing shared file: ${sharedFilePath}`,
      `Current generated source: ${sourceFilePath}`,
      'All targets in a shared dedupe group must normalize to identical support content.',
    ].join('\n'),
  );
}

function getTargetKeysForSharedOpenApiSupportGroup(sharedGroup) {
  return Object.keys(SPEC_TARGETS).filter((targetKey) => {
    const targetGroup = getSharedOpenApiSupportGroup(SPEC_TARGETS[targetKey]);
    return targetGroup && targetGroup.sharedRoot === sharedGroup.sharedRoot;
  });
}

function dedupeGeneratedOpenApiSupportFiles(repoRoot, targetKey) {
  const target = SPEC_TARGETS[targetKey];
  const sharedGroup = getSharedOpenApiSupportGroup(target);
  if (!sharedGroup) {
    return [];
  }

  const outputPath = path.join(repoRoot, target.output);
  const sharedRootPath = path.join(repoRoot, sharedGroup.sharedRoot);
  const sharedDirs = new Set();

  for (const supportFile of SHARED_OPENAPI_SUPPORT_FILES) {
    const sourceFilePath = path.join(outputPath, supportFile.relativePath);
    if (!fs.existsSync(sourceFilePath)) {
      continue;
    }

    const sharedFilePath = path.join(sharedRootPath, supportFile.relativePath);
    const sharedSource = createSharedOpenApiSupportSource(fs.readFileSync(sourceFilePath, 'utf8'));

    assertSharedOpenApiSupportMatches(sharedFilePath, sharedSource, sourceFilePath);
    writeFileIfChanged(sharedFilePath, sharedSource);
    writeFileIfChanged(
      sourceFilePath,
      createOpenApiReExportShim(sharedFilePath, sourceFilePath, sharedGroup.importExtension),
    );
    sharedDirs.add(sharedRootPath);
  }

  return [...sharedDirs];
}

function resetFullySelectedSharedOpenApiSupportDirs(repoRoot, targets) {
  const selectedTargets = new Set(targets);
  for (const sharedGroup of SHARED_OPENAPI_SUPPORT_GROUPS) {
    const groupTargetKeys = getTargetKeysForSharedOpenApiSupportGroup(sharedGroup);
    const selectedGroupTargetCount = groupTargetKeys.filter((targetKey) => selectedTargets.has(targetKey))
      .length;

    if (selectedGroupTargetCount === 0 || selectedGroupTargetCount !== groupTargetKeys.length) {
      continue;
    }

    fs.rmSync(path.join(repoRoot, sharedGroup.sharedRoot), {
      recursive: true,
      force: true,
    });
  }
}

function getSharedOpenApiSupportDirs(repoRoot, targets) {
  return [
    ...new Set(
      targets
        .map((targetKey) => {
          const sharedGroup = getSharedOpenApiSupportGroup(SPEC_TARGETS[targetKey]);
          return sharedGroup ? path.join(repoRoot, sharedGroup.sharedRoot) : null;
        })
        .filter(Boolean),
    ),
  ];
}

function dedupeTargetsOpenApiSupportFiles(repoRoot, targets) {
  const sharedDirs = new Set();
  for (const targetKey of targets) {
    for (const sharedDir of dedupeGeneratedOpenApiSupportFiles(repoRoot, targetKey)) {
      sharedDirs.add(sharedDir);
    }
  }
  return [...sharedDirs];
}

function listTypeScriptFiles(rootDir) {
  if (!fs.existsSync(rootDir)) {
    return [];
  }

  const files = [];
  const pending = [rootDir];
  while (pending.length > 0) {
    const currentDir = pending.pop();
    const entries = fs.readdirSync(currentDir, { withFileTypes: true });
    for (const entry of entries) {
      const entryPath = path.join(currentDir, entry.name);
      if (entry.isDirectory()) {
        pending.push(entryPath);
        continue;
      }
      if (entry.isFile() && entry.name.endsWith('.ts')) {
        files.push(entryPath);
      }
    }
  }
  return files;
}

function resolvePrettierModule(repoRoot) {
  try {
    const prettierPath = require.resolve('prettier', {
      paths: [path.join(repoRoot, 'frontend')],
    });
    return require(prettierPath);
  } catch (error) {
    fatal(`Prettier is required for API generation. Run \`cd frontend && npm ci\` and retry.`);
  }
}

async function formatGeneratedTypeScript(repoRoot, outputDirs) {
  const typeScriptFiles = [];
  for (const outputDir of outputDirs) {
    typeScriptFiles.push(...listTypeScriptFiles(outputDir));
  }
  if (typeScriptFiles.length === 0) {
    return;
  }
  const prettier = resolvePrettierModule(repoRoot);
  for (const filePath of typeScriptFiles) {
    const original = fs.readFileSync(filePath, 'utf8');
    const config = (await prettier.resolveConfig(filePath)) || {};
    const formatted = await Promise.resolve(
      prettier.format(original, {
        ...config,
        filepath: filePath,
      }),
    );
    if (formatted !== original) {
      fs.writeFileSync(filePath, formatted);
    }
  }
}

function isServerTarget(target) {
  return target.output.startsWith('frontend/server/src/generated/');
}

function buildDockerArgs(repoRoot, targetKey) {
  const target = SPEC_TARGETS[targetKey];
  const uid = process.getuid ? String(process.getuid()) : '1000';
  const gid = process.getgid ? String(process.getgid()) : '1000';
  const additionalProperties = [...BASE_ADDITIONAL_PROPERTIES];
  if (isServerTarget(target)) {
    additionalProperties.push('importFileExtension=.js');
  }

  return [
    'run',
    '--rm',
    '-u',
    `${uid}:${gid}`,
    '-v',
    `${repoRoot}:/local`,
    GENERATOR_IMAGE,
    'generate',
    '-i',
    `/local/${target.spec}`,
    '-g',
    'typescript-fetch',
    '-o',
    `/local/${target.output}`,
    '-c',
    '/local/frontend/swagger-config.json',
    '--remove-operation-id-prefix',
    '--additional-properties',
    additionalProperties.join(','),
    '--global-property',
    GLOBAL_PROPERTIES,
  ];
}

function prepareTarget(repoRoot, targetKey) {
  const target = SPEC_TARGETS[targetKey];
  const specPath = path.join(repoRoot, target.spec);
  const outputPath = path.join(repoRoot, target.output);
  const swaggerConfigPath = path.join(repoRoot, 'frontend/swagger-config.json');

  if (!fs.existsSync(specPath)) {
    fatal(`Spec not found: ${specPath}`);
  }
  if (!fs.existsSync(swaggerConfigPath)) {
    fatal(`Config not found: ${swaggerConfigPath}`);
  }

  fs.rmSync(outputPath, { recursive: true, force: true });
}

function postProcessTarget(repoRoot, targetKey) {
  const target = SPEC_TARGETS[targetKey];
  const outputPath = path.join(repoRoot, target.output);

  removeGeneratorMetadata(outputPath);
  normalizeGeneratedTypeScript(outputPath, {
    nodeCompatibleFetchTypes: isServerTarget(target),
    renameV1ApiSymbols: targetKey.startsWith('v1:'),
  });
}

function generateTarget(repoRoot, targetKey) {
  prepareTarget(repoRoot, targetKey);
  console.log(`Generating ${targetKey} -> ${SPEC_TARGETS[targetKey].output}`);
  runCommand('docker', buildDockerArgs(repoRoot, targetKey));
  postProcessTarget(repoRoot, targetKey);
}

async function runPool(taskFns, concurrency) {
  let nextIdx = 0;
  let failed = false;

  async function worker() {
    while (!failed && nextIdx < taskFns.length) {
      const idx = nextIdx++;
      try {
        await taskFns[idx]();
      } catch (err) {
        failed = true;
        throw err;
      }
    }
  }

  const results = await Promise.allSettled(
    Array.from({ length: Math.min(concurrency, taskFns.length) }, () => worker()),
  );
  const firstFailure = results.find((r) => r.status === 'rejected');
  if (firstFailure) {
    throw firstFailure.reason;
  }
}

async function generateTargetsParallel(repoRoot, targets, concurrency, hooks) {
  const {
    prepare = prepareTarget,
    dockerGenerate = (rr, tk) => runCommandAsync('docker', buildDockerArgs(rr, tk)),
    postProcess = postProcessTarget,
    pullImage = ensureImageAvailable,
  } = hooks || {};

  pullImage();

  const effectiveConcurrency = Math.min(concurrency, targets.length);
  console.log(`Generating ${targets.length} target(s) with concurrency ${effectiveConcurrency}...`);
  const startTime = Date.now();

  const tasks = targets.map((targetKey) => async () => {
    const target = SPEC_TARGETS[targetKey];
    prepare(repoRoot, targetKey);
    console.log(`  [start] ${targetKey} -> ${target.output}`);
    const taskStart = Date.now();
    try {
      await dockerGenerate(repoRoot, targetKey);
      postProcess(repoRoot, targetKey);
      const elapsed = ((Date.now() - taskStart) / 1000).toFixed(1);
      console.log(`  [done]  ${targetKey} (${elapsed}s)`);
    } catch (error) {
      console.error(`  [fail]  ${targetKey}`);
      if (error.stderr) {
        process.stderr.write(error.stderr);
      }
      throw error;
    }
  });

  await runPool(tasks, effectiveConcurrency);

  const totalElapsed = ((Date.now() - startTime) / 1000).toFixed(1);
  console.log(`Generated ${targets.length} target(s) in ${totalElapsed}s`);
}

async function main() {
  const scriptDir = __dirname;
  const repoRoot = path.resolve(scriptDir, '..', '..');
  const args = process.argv.slice(2);
  const targets = resolveTargets(args);
  const concurrency = Math.max(
    1,
    parseInt(process.env.OPENAPI_CONCURRENCY || '', 10) || DEFAULT_CONCURRENCY,
  );

  ensureDockerAvailable();

  if (targets.length === 1) {
    generateTarget(repoRoot, targets[0]);
  } else {
    await generateTargetsParallel(repoRoot, targets, concurrency);
  }

  const outputDirs = [...new Set(targets.map((targetKey) => path.join(repoRoot, SPEC_TARGETS[targetKey].output)))];
  await formatGeneratedTypeScript(repoRoot, outputDirs);

  resetFullySelectedSharedOpenApiSupportDirs(repoRoot, targets);
  const sharedDirs = dedupeTargetsOpenApiSupportFiles(repoRoot, targets);

  const allFormattedDirs = [...new Set([...outputDirs, ...sharedDirs, ...getSharedOpenApiSupportDirs(repoRoot, targets)])];
  await formatGeneratedTypeScript(repoRoot, allFormattedDirs);
}

if (require.main === module) {
  main().catch((error) => fatal(error.stack || error.message));
}

module.exports = {
  formatGeneratedTypeScript,
  assertSharedOpenApiSupportMatches,
  createOpenApiReExportShim,
  createSharedOpenApiSupportSource,
  dedupeGeneratedOpenApiSupportFiles,
  dedupeTargetsOpenApiSupportFiles,
  generateTargetsParallel,
  getSharedOpenApiSupportDirs,
  getTargetKeysForSharedOpenApiSupportGroup,
  listTypeScriptFiles,
  main,
  normalizeGeneratedTypeScript,
  normalizeGeneratedTypeScriptSource,
  normalizeNodeCompatibleFetchTypes,
  normalizeV1ApiSymbols,
  resetFullySelectedSharedOpenApiSupportDirs,
  resolvePrettierModule,
  resolveTargets,
  runPool,
};
