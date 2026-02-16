#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

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
  v1: Object.keys(SPEC_TARGETS).filter(key => key.startsWith('v1:')),
  v2beta1: Object.keys(SPEC_TARGETS).filter(key => key.startsWith('v2beta1:')),
  all: Object.keys(SPEC_TARGETS),
};

function fatal(message) {
  console.error(message);
  process.exit(1);
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

function runCommand(command, args) {
  const result = spawnSync(command, args, { stdio: 'inherit' });
  if (result.error) {
    fatal(`Failed to run ${command}: ${result.error.message}`);
  }
  if (result.status !== 0) {
    process.exit(result.status || 1);
  }
}

function ensureDockerAvailable() {
  const result = spawnSync('docker', ['--version'], { encoding: 'utf8' });
  if (result.error || result.status !== 0) {
    fatal(
      'Docker is required for API generation. Install/start Docker and retry `npm run apis`.',
    );
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

function normalizeV1ApiMethodNames(outputDir) {
  const apisDir = path.join(outputDir, 'apis');
  if (!fs.existsSync(apisDir)) {
    return;
  }

  const entries = fs.readdirSync(apisDir, { withFileTypes: true });
  for (const entry of entries) {
    if (!entry.isFile() || !entry.name.endsWith('.ts')) {
      continue;
    }
    const filePath = path.join(apisDir, entry.name);
    const original = fs.readFileSync(filePath, 'utf8');
    let updated = original;

    updated = updated.replace(/\b([A-Z][A-Za-z0-9_]*)V1Request\b/g, '$1Request');
    updated = updated.replace(/\b([a-z][A-Za-z0-9_]*)V1Raw\b/g, '$1Raw');
    updated = updated.replace(/\b([a-z][A-Za-z0-9_]*)V1\b/g, '$1');

    if (updated !== original) {
      fs.writeFileSync(filePath, updated);
    }
  }
}

function normalizeGeneratedTypeScriptForLegacyTooling(outputDir, options = {}) {
  if (!fs.existsSync(outputDir)) {
    return;
  }

  const { nodeCompatibleFetchTypes = false } = options;
  const tsFiles = listTypeScriptFiles(outputDir);
  for (const filePath of tsFiles) {
    const original = fs.readFileSync(filePath, 'utf8');
    let updated = original;

    updated = updated.replace(/import\s+type\s+([^;]+;)/g, 'import $1');
    updated = updated.replace(/\boverride\s+([A-Za-z_$][A-Za-z0-9_$]*\s*:)/g, '$1');
    if (nodeCompatibleFetchTypes) {
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
    }

    if (updated !== original) {
      fs.writeFileSync(filePath, updated);
    }
  }
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

function formatGeneratedTypeScript(outputDirs) {
  const typeScriptFiles = [];
  for (const outputDir of outputDirs) {
    typeScriptFiles.push(...listTypeScriptFiles(outputDir));
  }
  if (typeScriptFiles.length === 0) {
    return;
  }
  runCommand('prettier', ['--write', ...typeScriptFiles]);
}

function generateTarget(repoRoot, targetKey) {
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

  const uid = process.getuid ? String(process.getuid()) : '1000';
  const gid = process.getgid ? String(process.getgid()) : '1000';
  const additionalProperties = [...BASE_ADDITIONAL_PROPERTIES];
  const isServerTarget = target.output.startsWith('frontend/server/src/generated/');
  if (isServerTarget) {
    additionalProperties.push('importFileExtension=.js');
  }

  const dockerArgs = [
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

  console.log(`Generating ${targetKey} -> ${target.output}`);
  runCommand('docker', dockerArgs);
  removeGeneratorMetadata(outputPath);
  if (targetKey.startsWith('v1:')) {
    normalizeV1ApiMethodNames(outputPath);
  }
  normalizeGeneratedTypeScriptForLegacyTooling(outputPath, {
    nodeCompatibleFetchTypes: isServerTarget,
  });
}

function main() {
  const scriptDir = __dirname;
  const repoRoot = path.resolve(scriptDir, '..', '..');
  const args = process.argv.slice(2);
  const targets = resolveTargets(args);

  ensureDockerAvailable();
  for (const targetKey of targets) {
    generateTarget(repoRoot, targetKey);
  }
  const outputDirs = [...new Set(targets.map(targetKey => path.join(repoRoot, SPEC_TARGETS[targetKey].output)))];
  formatGeneratedTypeScript(outputDirs);
}

main();
