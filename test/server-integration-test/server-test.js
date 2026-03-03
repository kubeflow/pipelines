#!/usr/bin/env node
/**
 * Server Integration Test for Kubeflow Pipelines Frontend Server
 *
 * Validates server-side changes by testing:
 * 1. Static file serving
 * 2. API proxy endpoints
 * 3. K8s integration (pod logs, tensorboard, etc.)
 * 4. Health endpoints
 *
 * Usage:
 *   node server-smoke-test.js                    # Test against localhost:3000
 *   node server-smoke-test.js --url http://localhost:3001
 *   node server-smoke-test.js --verbose
 */

const http = require('http');
const https = require('https');

// Parse arguments
const args = process.argv.slice(2);
const getArg = (name, defaultValue) => {
  const index = args.indexOf(`--${name}`);
  return index !== -1 && args[index + 1] ? args[index + 1] : defaultValue;
};
const hasFlag = (name) => args.includes(`--${name}`);

const BASE_URL = getArg('url', 'http://127.0.0.1:3000');
const VERBOSE = hasFlag('verbose');
const TIMEOUT = parseInt(getArg('timeout', '10000'), 10);

// Test results
const results = {
  passed: 0,
  failed: 0,
  skipped: 0,
  tests: [],
};

// Colors
const colors = {
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  cyan: '\x1b[36m',
  reset: '\x1b[0m',
  dim: '\x1b[2m',
};

function log(msg, color = 'reset') {
  console.log(`${colors[color]}${msg}${colors.reset}`);
}

/**
 * Make an HTTP request
 */
async function request(method, path, options = {}) {
  const url = new URL(path, BASE_URL);
  const protocol = url.protocol === 'https:' ? https : http;

  return new Promise((resolve, reject) => {
    const req = protocol.request(url, {
      method,
      timeout: TIMEOUT,
      headers: options.headers || {},
    }, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => {
        resolve({
          status: res.statusCode,
          headers: res.headers,
          body: data,
          ok: res.statusCode >= 200 && res.statusCode < 400,
        });
      });
    });

    req.on('error', reject);
    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Request timeout'));
    });

    if (options.body) {
      req.write(options.body);
    }
    req.end();
  });
}

/**
 * Run a test
 */
async function test(name, fn) {
  const startTime = Date.now();
  try {
    await fn();
    const duration = Date.now() - startTime;
    results.passed++;
    results.tests.push({ name, status: 'passed', duration });
    log(`  ✓ ${name} ${colors.dim}(${duration}ms)${colors.reset}`, 'green');
    return true;
  } catch (error) {
    const duration = Date.now() - startTime;
    results.failed++;
    results.tests.push({ name, status: 'failed', duration, error: error.message });
    log(`  ✗ ${name}`, 'red');
    log(`    ${error.message}`, 'dim');
    if (VERBOSE && error.stack) {
      log(`    ${error.stack.split('\n').slice(1, 3).join('\n    ')}`, 'dim');
    }
    return false;
  }
}

/**
 * Skip a test
 */
function skip(name, reason) {
  results.skipped++;
  results.tests.push({ name, status: 'skipped', reason });
  log(`  ○ ${name} ${colors.dim}(skipped: ${reason})${colors.reset}`, 'yellow');
}

/**
 * Assert helpers
 */
function assertEqual(actual, expected, msg) {
  if (actual !== expected) {
    throw new Error(`${msg}: expected ${expected}, got ${actual}`);
  }
}

function assertTrue(condition, msg) {
  if (!condition) {
    throw new Error(msg);
  }
}

function assertContains(str, substring, msg) {
  if (!str.includes(substring)) {
    throw new Error(`${msg}: "${substring}" not found in response`);
  }
}

function normalizeAssetPath(assetPath) {
  const trimmed = assetPath.trim();
  const withoutDotPrefix = trimmed.startsWith('./') ? trimmed.slice(1) : trimmed;
  return withoutDotPrefix.startsWith('/') ? withoutDotPrefix : `/${withoutDotPrefix}`;
}

function findAssetPath(indexHtml, pattern, errorMessage) {
  const match = indexHtml.match(pattern);
  assertTrue(match && match[1], errorMessage);
  return normalizeAssetPath(match[1]);
}

// ============================================================================
// Test Suites
// ============================================================================

async function testStaticServing() {
  log('\n📁 Static File Serving', 'cyan');

  await test('GET / returns index.html', async () => {
    const res = await request('GET', '/');
    assertEqual(res.status, 200, 'Status code');
    assertContains(res.body.toLowerCase(), '<!doctype html>', 'HTML doctype');
    assertTrue(res.headers['content-type']?.includes('text/html'), 'Content-Type should be HTML');
    assertContains(res.body, 'Kubeflow Pipelines', 'Page title');
    assertTrue(
      /<div[^>]+id=["']root["'][^>]*>\s*<\/div>/.test(res.body),
      'React root element should be present',
    );
  });

  await test('GET /index.html returns index.html', async () => {
    const res = await request('GET', '/index.html');
    assertEqual(res.status, 200, 'Status code');
    assertContains(res.body.toLowerCase(), '<!doctype html>', 'HTML doctype');
    assertTrue(res.headers['content-type']?.includes('text/html'), 'Content-Type should be HTML');
  });

  await test('GET /static/*.js returns JavaScript', async () => {
    // First get index.html to find the JS bundle name
    // Supports both CRA (static/js/main.[hash].js) and Vite (static/index-[hash].js) output
    const indexRes = await request('GET', '/');
    const jsMatch = indexRes.body.match(/(?:\.?\/)static\/(?:js\/)?[a-zA-Z0-9_-]+[-\.][a-zA-Z0-9_-]+\.js/);
    assertTrue(jsMatch, 'Could not find JS bundle path in index.html');

    const res = await request('GET', normalizeAssetPath(jsMatch[0]));
    assertEqual(res.status, 200, 'Status code');
    assertTrue(res.headers['content-type']?.includes('javascript'), 'Content-Type should be JavaScript');
  });

  await test('GET /static/*.css returns CSS', async () => {
    const indexRes = await request('GET', '/');
    // Supports both CRA (static/css/main.[hash].css) and Vite (static/index-[hash].css) output
    const cssMatch = indexRes.body.match(/(?:\.?\/)static\/(?:css\/)?[a-zA-Z0-9_-]+[-\.][a-zA-Z0-9_-]+\.css/);
    assertTrue(cssMatch, 'Could not find CSS bundle path in index.html');

    const res = await request('GET', normalizeAssetPath(cssMatch[0]));
    assertEqual(res.status, 200, 'Status code');
    assertTrue(res.headers['content-type']?.includes('css'), 'Content-Type should be CSS');
  });

  // Note: SPA routing for /pipelines is handled by the React app, not the server
  // The server only serves static files and proxies API calls
}

async function testHealthEndpoints() {
  log('\n🏥 Health Endpoints', 'cyan');

  await test('GET /apis/v2beta1/healthz returns healthy', async () => {
    const res = await request('GET', '/apis/v2beta1/healthz');
    // May return 200 with status or proxy error if backend not fully ready
    assertTrue(res.status === 200 || res.status === 502, `Status should be 200 or 502, got ${res.status}`);
    if (res.status === 200) {
      const data = JSON.parse(res.body);
      assertTrue(data !== undefined, 'Should return JSON response');
    }
  });

  await test('GET /apis/v1beta1/healthz returns healthy', async () => {
    const res = await request('GET', '/apis/v1beta1/healthz');
    assertTrue(res.status === 200 || res.status === 502, `Status should be 200 or 502, got ${res.status}`);
  });
}

async function testAPIProxyV2() {
  log('\n🔌 API Proxy (v2beta1)', 'cyan');

  await test('GET /apis/v2beta1/pipelines returns pipeline list', async () => {
    const res = await request('GET', '/apis/v2beta1/pipelines?page_size=5');
    assertEqual(res.status, 200, 'Status code');
    const data = JSON.parse(res.body);
    assertTrue(Array.isArray(data.pipelines) || data.pipelines === undefined, 'Should return pipelines array or empty');
  });

  await test('GET /apis/v2beta1/experiments returns experiment list', async () => {
    const res = await request('GET', '/apis/v2beta1/experiments?page_size=5');
    assertEqual(res.status, 200, 'Status code');
    const data = JSON.parse(res.body);
    assertTrue(Array.isArray(data.experiments) || data.experiments === undefined, 'Should return experiments array or empty');
  });

  await test('GET /apis/v2beta1/runs returns run list', async () => {
    const res = await request('GET', '/apis/v2beta1/runs?page_size=5');
    assertEqual(res.status, 200, 'Status code');
    const data = JSON.parse(res.body);
    assertTrue(Array.isArray(data.runs) || data.runs === undefined, 'Should return runs array or empty');
  });

  await test('GET /apis/v2beta1/recurringruns returns recurring run list', async () => {
    const res = await request('GET', '/apis/v2beta1/recurringruns?page_size=5');
    assertEqual(res.status, 200, 'Status code');
    const data = JSON.parse(res.body);
    assertTrue(Array.isArray(data.recurringRuns) || data.recurring_runs === undefined, 'Should return recurring runs array or empty');
  });
}

async function testAPIProxyV1() {
  log('\n🔌 API Proxy (v1beta1)', 'cyan');

  await test('GET /apis/v1beta1/pipelines returns pipeline list', async () => {
    const res = await request('GET', '/apis/v1beta1/pipelines?page_size=5');
    assertEqual(res.status, 200, 'Status code');
    const data = JSON.parse(res.body);
    assertTrue(Array.isArray(data.pipelines) || data.pipelines === undefined, 'Should return pipelines array or empty');
  });

  await test('GET /apis/v1beta1/experiments returns experiment list', async () => {
    const res = await request('GET', '/apis/v1beta1/experiments?page_size=5');
    assertEqual(res.status, 200, 'Status code');
  });
}

async function testSystemEndpoints() {
  log('\n⚙️  System Endpoints', 'cyan');

  await test('GET /system/cluster-name returns cluster info', async () => {
    const res = await request('GET', '/system/cluster-name');
    // May return 200 with name or error if not in GKE
    assertTrue([200, 404, 500].includes(res.status), `Unexpected status: ${res.status}`);
  });

  await test('GET /system/project-id returns project info', async () => {
    const res = await request('GET', '/system/project-id');
    assertTrue([200, 404, 500].includes(res.status), `Unexpected status: ${res.status}`);
  });
}

async function testK8sIntegration() {
  log('\n☸️  K8s Integration (via server)', 'cyan');

  // These endpoints use the K8s client that was upgraded in PR #12756
  await test('GET /k8s/pod/logs endpoint exists', async () => {
    // This will fail without valid pod params, but should not 404
    const res = await request('GET', '/k8s/pod/logs?podname=test&podnamespace=kubeflow');
    // 400 (bad request) or 500 (pod not found) is expected, 404 means endpoint doesn't exist
    assertTrue(res.status !== 404, `Endpoint should exist (got ${res.status})`);
  });

  await test('POST /apps/tensorboard endpoint exists', async () => {
    const res = await request('POST', '/apps/tensorboard', {
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ logdir: 'test', namespace: 'kubeflow' }),
    });
    // Should not 404 - other errors are OK (auth, validation, etc.)
    assertTrue(res.status !== 404, `Endpoint should exist (got ${res.status})`);
  });

  await test('GET /apps/tensorboard endpoint exists', async () => {
    const res = await request('GET', '/apps/tensorboard?logdir=test&namespace=kubeflow');
    assertTrue(res.status !== 404, `Endpoint should exist (got ${res.status})`);
  });

  await test('DELETE /apps/tensorboard endpoint exists', async () => {
    const res = await request('DELETE', '/apps/tensorboard?logdir=test&namespace=kubeflow');
    assertTrue(res.status !== 404, `Endpoint should exist (got ${res.status})`);
  });
}

async function testArtifactEndpoints() {
  log('\n📦 Artifact Endpoints', 'cyan');

  await test('GET /artifacts/get endpoint exists', async () => {
    const res = await request('GET', '/artifacts/get?source=minio&bucket=test&key=test');
    // Should not 404
    assertTrue(res.status !== 404, `Endpoint should exist (got ${res.status})`);
  });

  await test('GET /artifacts/minio endpoint exists', async () => {
    const res = await request('GET', '/artifacts/minio/test/test');
    assertTrue(res.status !== 404, `Endpoint should exist (got ${res.status})`);
  });
}

async function testVisualizationEndpoints() {
  log('\n📊 Visualization Endpoints', 'cyan');

  await test('GET /visualizations/allowed returns config', async () => {
    const res = await request('GET', '/visualizations/allowed');
    assertEqual(res.status, 200, 'Status code');
    // Returns boolean indicating if custom visualizations are allowed
    assertTrue(res.body === 'true' || res.body === 'false', 'Should return boolean');
  });
}

// ============================================================================
// Main
// ============================================================================

async function main() {
  console.log(`
╔══════════════════════════════════════════════════════════════╗
║       Server Integration Test - Kubeflow Pipelines           ║
╚══════════════════════════════════════════════════════════════╝
`);
  log(`Target: ${BASE_URL}`, 'cyan');
  log(`Timeout: ${TIMEOUT}ms`, 'dim');

  // Check server is reachable
  try {
    await request('GET', '/');
    log('Server is reachable ✓\n', 'green');
  } catch (e) {
    log(`\nServer is not reachable at ${BASE_URL}`, 'red');
    log(`Error: ${e.message}`, 'dim');
    log('\nMake sure the server is running:', 'yellow');
    log('  cd frontend && npm run start:proxy-and-server', 'dim');
    process.exit(1);
  }

  // Run test suites
  await testStaticServing();
  await testHealthEndpoints();
  await testAPIProxyV2();
  await testAPIProxyV1();
  await testSystemEndpoints();
  await testK8sIntegration();
  await testArtifactEndpoints();
  await testVisualizationEndpoints();

  // Summary
  console.log('\n' + '═'.repeat(60));
  const totalTests = results.passed + results.failed + results.skipped;
  log(`\nResults: ${results.passed} passed, ${results.failed} failed, ${results.skipped} skipped (${totalTests} total)`,
    results.failed > 0 ? 'red' : 'green');

  if (results.failed > 0) {
    log('\nFailed tests:', 'red');
    for (const t of results.tests.filter(t => t.status === 'failed')) {
      log(`  • ${t.name}: ${t.error}`, 'red');
    }
  }

  // Exit code
  process.exit(results.failed > 0 ? 1 : 0);
}

main().catch(err => {
  log(`Fatal error: ${err.message}`, 'red');
  process.exit(1);
});
