#!/usr/bin/env node
/**
 * Change Detection for UI Smoke Tests
 *
 * Maps git diff output to backend components, determining which
 * Docker images need to be rebuilt and which K8s deployments
 * need to be restarted.
 *
 * Usage:
 *   node detect-changes.js --base master
 *   node detect-changes.js --base release
 *   node detect-changes.js --base 2.15.0
 */

const { execSync } = require('child_process');

/**
 * Component definitions mapping source paths to build/deploy targets.
 *
 * Each component has:
 *   - paths: file path prefixes that indicate this component changed
 *   - makeTarget: the `make -C backend <target>` to build the Docker image
 *   - imageTag: the Docker image tag (matches IMG_TAG_* in backend/Makefile)
 *   - deployment: the K8s deployment name to restart (null for runtime-only images)
 */
const COMPONENTS = [
  {
    name: 'apiserver',
    paths: ['backend/src/apiserver/'],
    // Exclude visualization subdirectory — it's a separate component
    excludePaths: ['backend/src/apiserver/visualization/'],
    makeTarget: 'image_apiserver',
    imageTag: 'apiserver',
    deployment: 'ml-pipeline',
  },
  {
    name: 'persistence-agent',
    paths: ['backend/src/agent/persistence/'],
    makeTarget: 'image_persistence_agent',
    imageTag: 'persistence-agent',
    deployment: 'ml-pipeline-persistenceagent',
  },
  {
    name: 'cache-server',
    paths: ['backend/src/cache/'],
    makeTarget: 'image_cache',
    imageTag: 'cache-server',
    deployment: null, // cache deployment name varies
  },
  {
    name: 'scheduledworkflow',
    paths: ['backend/src/crd/controller/scheduledworkflow/'],
    makeTarget: 'image_swf',
    imageTag: 'scheduledworkflow',
    deployment: 'ml-pipeline-scheduledworkflow',
  },
  {
    name: 'viewercontroller',
    paths: ['backend/src/crd/controller/viewer/'],
    makeTarget: 'image_viewer',
    imageTag: 'viewercontroller',
    deployment: 'ml-pipeline-viewer-crd',
  },
  {
    name: 'visualization',
    paths: ['backend/src/apiserver/visualization/'],
    makeTarget: 'image_visualization',
    imageTag: 'visualization',
    deployment: 'ml-pipeline-visualizationserver',
  },
  {
    name: 'driver',
    paths: ['backend/src/v2/cmd/driver/'],
    makeTarget: 'image_driver',
    imageTag: 'kfp-driver',
    deployment: null, // runtime image, no standing deployment
  },
  {
    name: 'launcher',
    paths: ['backend/src/v2/cmd/launcher-v2/'],
    makeTarget: 'image_launcher',
    imageTag: 'kfp-launcher',
    deployment: null, // runtime image, no standing deployment
  },
];

// Paths that affect ALL Go-based backend components
const GLOBAL_BACKEND_PATHS = [
  'backend/src/common/',
  'go.mod',
  'go.sum',
];

/**
 * Get the latest release tag (semantic versioning).
 * Returns null if no tags found.
 */
function getLatestRelease() {
  try {
    const output = execSync('git tag --list', { encoding: 'utf8' });
    const tags = output
      .split('\n')
      .filter(t => /^\d+\.\d+\.\d+$/.test(t.trim()))
      .map(t => t.trim())
      .filter(Boolean);

    if (tags.length === 0) return null;

    // Sort by semver: split into [major, minor, patch] and compare numerically
    tags.sort((a, b) => {
      const pa = a.split('.').map(Number);
      const pb = b.split('.').map(Number);
      for (let i = 0; i < 3; i++) {
        if (pa[i] !== pb[i]) return pa[i] - pb[i];
      }
      return 0;
    });

    return tags[tags.length - 1];
  } catch (e) {
    return null;
  }
}

/**
 * Resolve a base ref string to an actual git ref.
 *   "master" → "master"
 *   "release" → latest semver tag (e.g. "2.15.0")
 *   anything else → passed through as-is
 */
function resolveBaseRef(ref) {
  if (ref === 'release') {
    const latest = getLatestRelease();
    if (!latest) {
      throw new Error('No release tags found. Use a specific ref instead.');
    }
    return latest;
  }
  return ref;
}

/**
 * Get list of changed files between a base ref and a head ref.
 *
 * Uses 2-dot diff (baseRef..headRef) which shows only what headRef added —
 * i.e. the actual PR changes. 3-dot diff (baseRef...headRef) would also
 * include files changed on the base branch since divergence, causing
 * unnecessary backend rebuilds for frontend-only PRs.
 *
 * @param {string} baseRef - Base git ref (e.g. "master", "2.15.0")
 * @param {string} [headRef='HEAD'] - Head git ref (e.g. "HEAD", "pr-12756")
 */
function getChangedFiles(baseRef, headRef = 'HEAD') {
  try {
    const output = execSync(`git diff --name-only ${baseRef}..${headRef}`, {
      encoding: 'utf8',
    });
    return output.split('\n').filter(Boolean);
  } catch (e) {
    throw new Error(`Failed to diff ${baseRef}..${headRef}: ${e.message}`);
  }
}

/**
 * Check if a file path matches a component, respecting excludePaths.
 */
function fileMatchesComponent(file, component) {
  const matchesInclude = component.paths.some(p => file.startsWith(p));
  if (!matchesInclude) return false;
  if (component.excludePaths) {
    return !component.excludePaths.some(p => file.startsWith(p));
  }
  return true;
}

/**
 * Detect which components changed between baseRef and headRef.
 *
 * @param {string} baseRef - Base ref string ("master", "release", or a tag/SHA)
 * @param {string} [headRef='HEAD'] - Head ref (e.g. "HEAD", "pr-12756")
 *
 * Returns:
 *   {
 *     baseRef: string,           // resolved ref (e.g. "2.15.0" if "release" was passed)
 *     headRef: string,           // head ref used for the diff
 *     changedFiles: string[],    // all changed file paths
 *     components: object[],      // COMPONENTS entries that need rebuilding
 *     frontendChanged: boolean,  // frontend/src/** changed
 *     serverChanged: boolean,    // frontend/server/** changed
 *     backendChanged: boolean,   // any backend component changed
 *     manifestsChanged: boolean, // manifests/** changed
 *   }
 */
function detectChanges(baseRef, headRef = 'HEAD') {
  const resolved = resolveBaseRef(baseRef);
  const changedFiles = getChangedFiles(resolved, headRef);

  // Check for global backend changes (common/, go.mod, go.sum)
  const globalBackendChanged = changedFiles.some(f =>
    GLOBAL_BACKEND_PATHS.some(p => f.startsWith(p) || f === p),
  );

  // Find specifically changed components
  const changedComponentNames = new Set();

  for (const file of changedFiles) {
    for (const component of COMPONENTS) {
      if (fileMatchesComponent(file, component)) {
        changedComponentNames.add(component.name);
      }
    }
  }

  // If global paths changed, all Go components are affected
  if (globalBackendChanged) {
    for (const component of COMPONENTS) {
      changedComponentNames.add(component.name);
    }
  }

  const components = COMPONENTS.filter(c => changedComponentNames.has(c.name));

  return {
    baseRef: resolved,
    headRef,
    changedFiles,
    components,
    frontendChanged: changedFiles.some(f => f.startsWith('frontend/src/')),
    serverChanged: changedFiles.some(f => f.startsWith('frontend/server/')),
    backendChanged: components.length > 0,
    manifestsChanged: changedFiles.some(f => f.startsWith('manifests/')),
  };
}

// CLI interface
if (require.main === module) {
  const args = process.argv.slice(2);
  const getArg = (name, defaultValue) => {
    const index = args.indexOf(`--${name}`);
    return index !== -1 && args[index + 1] ? args[index + 1] : defaultValue;
  };

  const baseRef = getArg('base', 'master');

  try {
    const result = detectChanges(baseRef);

    console.log(`Base ref: ${result.baseRef}`);
    console.log(`Changed files: ${result.changedFiles.length}`);
    console.log(`Frontend changed: ${result.frontendChanged}`);
    console.log(`Server changed: ${result.serverChanged}`);
    console.log(`Backend changed: ${result.backendChanged}`);
    console.log(`Manifests changed: ${result.manifestsChanged}`);

    if (result.components.length > 0) {
      console.log('\nBackend components to rebuild:');
      for (const c of result.components) {
        console.log(`  ${c.name} (make ${c.makeTarget})`);
        if (c.deployment) {
          console.log(`    → restart deployment/${c.deployment}`);
        }
      }
    } else {
      console.log('\nNo backend components to rebuild.');
    }
  } catch (e) {
    console.error(`Error: ${e.message}`);
    process.exit(1);
  }
}

module.exports = {
  COMPONENTS,
  detectChanges,
  getChangedFiles,
  getLatestRelease,
  resolveBaseRef,
};
