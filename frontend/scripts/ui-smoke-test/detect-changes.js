#!/usr/bin/env node
/**
 * Maps a PR diff to the backend images needed by the UI smoke test.
 */

const { execFileSync } = require('child_process');

const COMPONENTS = [
  {
    name: 'frontend',
    dockerfile: 'frontend/Dockerfile',
    imageTag: 'kfp-ui-smoke/frontend',
    deployment: 'ml-pipeline-ui',
    container: 'ml-pipeline-ui',
    buildArgs: {
      COMMIT_HASH: 'commitSha',
      TAG_NAME: 'tagName',
      DATE: 'buildDate',
      NODE_VERSION: 'nodeVersion',
    },
  },
  {
    name: 'apiserver',
    dockerfile: 'backend/Dockerfile',
    imageTag: 'kfp-ui-smoke/apiserver',
    deployment: 'ml-pipeline',
    container: 'ml-pipeline-api-server',
    buildArgs: {
      COMMIT_SHA: 'commitSha',
      TAG_NAME: 'tagName',
    },
  },
  {
    name: 'persistence-agent',
    dockerfile: 'backend/Dockerfile.persistenceagent',
    imageTag: 'kfp-ui-smoke/persistence-agent',
    deployment: 'ml-pipeline-persistenceagent',
    container: 'ml-pipeline-persistenceagent',
  },
  {
    name: 'cache-server',
    dockerfile: 'backend/Dockerfile.cacheserver',
    imageTag: 'kfp-ui-smoke/cache-server',
    deployment: 'cache-server',
    container: 'server',
  },
  {
    name: 'scheduledworkflow',
    dockerfile: 'backend/Dockerfile.scheduledworkflow',
    imageTag: 'kfp-ui-smoke/scheduledworkflow',
    deployment: 'ml-pipeline-scheduledworkflow',
    container: 'ml-pipeline-scheduledworkflow',
  },
  {
    name: 'viewercontroller',
    dockerfile: 'backend/Dockerfile.viewercontroller',
    imageTag: 'kfp-ui-smoke/viewercontroller',
    deployment: 'ml-pipeline-viewer-crd',
    container: 'ml-pipeline-viewer-crd',
  },
  {
    name: 'visualization',
    dockerfile: 'backend/Dockerfile.visualization',
    imageTag: 'kfp-ui-smoke/visualization',
    deployment: 'ml-pipeline-visualizationserver',
    container: 'ml-pipeline-visualizationserver',
    crossRevisionBuildInputs: [
      'backend/Dockerfile.visualization',
      'backend/src/apiserver/visualization',
    ],
  },
  {
    name: 'driver',
    dockerfile: 'backend/Dockerfile.driver',
    imageTag: 'kfp-ui-smoke/driver',
    deployment: null,
    container: null,
    runtimeEnv: 'V2_DRIVER_IMAGE',
  },
  {
    name: 'launcher',
    dockerfile: 'backend/Dockerfile.launcher',
    imageTag: 'kfp-ui-smoke/launcher',
    deployment: null,
    container: null,
    runtimeEnv: 'V2_LAUNCHER_IMAGE',
  },
  {
    name: 'metadata-writer',
    dockerfile: 'backend/metadata_writer/Dockerfile',
    imageTag: 'kfp-ui-smoke/metadata-writer',
    deployment: 'metadata-writer',
    container: 'main',
  },
  {
    name: 'cache-deployer',
    dockerfile: 'backend/src/cache/deployer/Dockerfile',
    imageTag: 'kfp-ui-smoke/cache-deployer',
    deployment: 'cache-deployer-deployment',
    container: 'main',
  },
  {
    name: 'metadata-envoy',
    dockerfile: 'third_party/metadata_envoy/Dockerfile',
    imageTag: 'kfp-ui-smoke/metadata-envoy',
    deployment: 'metadata-envoy-deployment',
    container: 'container',
  },
];

function git(args, cwd = process.cwd(), options = {}) {
  const output = execFileSync('git', args, {
    cwd,
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
  });
  return options.trim === false ? output : output.trim();
}

function parseFileList(output) {
  return output
    ? output
        .split('\n')
        .map((file) => file.trim())
        .filter(Boolean)
    : [];
}

function parseNullDelimitedFileList(output) {
  return output ? output.split('\0').filter((file) => file.length > 0) : [];
}

function validateGitRef(ref, options = {}) {
  const cwd = typeof options === 'string' ? options : options.cwd || process.cwd();
  if (typeof ref !== 'string' || ref.length === 0 || ref.startsWith('-')) {
    throw new Error(`Invalid git ref: ${JSON.stringify(ref)}`);
  }
  try {
    git(['rev-parse', '--verify', '--end-of-options', `${ref}^{commit}`], cwd);
    return ref;
  } catch (error) {
    throw new Error(`Git ref does not resolve to a commit: ${ref}`);
  }
}

function getLatestRelease(options = {}) {
  const { cwd = process.cwd() } = options;
  try {
    const tags = parseFileList(git(['tag', '--list'], cwd))
      .filter((tag) => /^\d+\.\d+\.\d+$/.test(tag))
      .sort((left, right) => {
        const leftParts = left.split('.').map(Number);
        const rightParts = right.split('.').map(Number);
        for (let index = 0; index < 3; index++) {
          if (leftParts[index] !== rightParts[index]) {
            return leftParts[index] - rightParts[index];
          }
        }
        return 0;
      });
    return tags.at(-1) || null;
  } catch (error) {
    return null;
  }
}

function resolveBaseRef(ref, options = {}) {
  if (ref !== 'release') return ref;
  const latest = getLatestRelease(options);
  if (!latest) {
    throw new Error('No release tags found. Use a specific ref instead.');
  }
  return latest;
}

function getWorkingTreeFiles(options = {}) {
  const { cwd = process.cwd() } = options;
  const files = [
    ...parseNullDelimitedFileList(
      git(['diff', '--no-renames', '--name-only', '-z', '--diff-filter=ACMRTD'], cwd, {
        trim: false,
      }),
    ),
    ...parseNullDelimitedFileList(
      git(['diff', '--cached', '--no-renames', '--name-only', '-z', '--diff-filter=ACMRTD'], cwd, {
        trim: false,
      }),
    ),
    ...parseNullDelimitedFileList(
      git(['ls-files', '--others', '--exclude-standard', '-z'], cwd, { trim: false }),
    ),
  ];
  return [...new Set(files)].sort();
}

/**
 * Return the files introduced on head since its merge base with base.
 * Dirty files may only be included when head is the checked-out HEAD.
 */
function getChangedFiles(baseRef, headRef = 'HEAD', options = {}) {
  const { cwd = process.cwd(), includeWorkingTree = false } = options;
  validateGitRef(baseRef, { cwd });
  validateGitRef(headRef, { cwd });

  if (includeWorkingTree && headRef !== 'HEAD') {
    throw new Error('Working-tree changes can only be included when headRef is HEAD.');
  }

  try {
    const committed = parseNullDelimitedFileList(
      git(
        [
          'diff',
          '--no-renames',
          '--name-only',
          '-z',
          '--diff-filter=ACMRTD',
          `${baseRef}...${headRef}`,
        ],
        cwd,
        { trim: false },
      ),
    );
    const dirty = includeWorkingTree ? getWorkingTreeFiles({ cwd }) : [];
    return [...new Set([...committed, ...dirty])].sort();
  } catch (error) {
    throw new Error(`Failed to diff ${baseRef}...${headRef}: ${error.message}`);
  }
}

function isBackendBuildInput(file) {
  return (
    file.startsWith('backend/') ||
    file.startsWith('api/') ||
    file.startsWith('kubernetes_platform/') ||
    file.startsWith('samples/') ||
    file.startsWith('third_party/metadata_envoy/') ||
    file === 'third_party/license.txt' ||
    file === 'go.mod' ||
    file === 'go.sum' ||
    file === '.dockerignore' ||
    file.split('/').at(-1).startsWith('Dockerfile')
  );
}

function detectChanges(baseRef, headRef = 'HEAD', options = {}) {
  const { cwd = process.cwd(), includeWorkingTree = false } = options;
  const resolved = resolveBaseRef(baseRef, { cwd });
  const changedFiles = getChangedFiles(resolved, headRef, { cwd, includeWorkingTree });
  const backendChanged = changedFiles.some(isBackendBuildInput);

  return {
    baseRef: resolved,
    headRef,
    changedFiles,
    components: backendChanged ? [...COMPONENTS] : [],
    frontendChanged: changedFiles.some((file) => file.startsWith('frontend/')),
    serverChanged: changedFiles.some((file) => file.startsWith('frontend/server/')),
    backendChanged,
    manifestsChanged: changedFiles.some((file) => file.startsWith('manifests/')),
  };
}

if (require.main === module) {
  const args = process.argv.slice(2);
  const valueFor = (name, fallback) => {
    const index = args.indexOf(`--${name}`);
    return index >= 0 && args[index + 1] ? args[index + 1] : fallback;
  };

  try {
    const result = detectChanges(valueFor('base', 'master'), valueFor('head', 'HEAD'), {
      includeWorkingTree: args.includes('--include-working-tree'),
    });
    console.log(JSON.stringify(result, null, 2));
  } catch (error) {
    console.error(`Error: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = {
  COMPONENTS,
  detectChanges,
  getChangedFiles,
  getLatestRelease,
  getWorkingTreeFiles,
  isBackendBuildInput,
  resolveBaseRef,
  validateGitRef,
};
