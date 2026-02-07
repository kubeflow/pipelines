# Server Integration Tests

Integration tests for the Kubeflow Pipelines frontend server. These tests validate server functionality against a running KFP deployment.

## Prerequisites

- Node.js 18+
- Running KFP deployment (Kind cluster or remote)
- Server accessible at `http://127.0.0.1:3000` (or specify with `--url`)

## Quick Start

```bash
# Start Kind cluster with KFP (if not already running)
make -C ../../backend kind-cluster-agnostic

# Run all tests
npm run test:all

# Or run directly
node server-test.js --url http://127.0.0.1:3000
node k8s-integration-test.js --url http://127.0.0.1:3000
```

## Tests

### server-test.js

General server functionality tests:
- Static file serving
- API proxy (v1beta1, v2beta1)
- Health endpoints
- System endpoints
- Artifact endpoints

```bash
node server-test.js --url http://127.0.0.1:3000
node server-test.js --url http://127.0.0.1:3000 --verbose
```

### k8s-integration-test.js

K8s client integration tests (validates `@kubernetes/client-node` upgrade):
- Pod log retrieval (`readNamespacedPodLog`)
- Tensorboard CRUD operations (`createNamespacedPod`, `deleteNamespacedPod`)
- Error handling with K8s client
- Namespace operations

```bash
node k8s-integration-test.js --url http://127.0.0.1:3000
node k8s-integration-test.js --url http://127.0.0.1:3000 --namespace kubeflow --verbose
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `--url` | `http://127.0.0.1:3000` | Server URL to test |
| `--namespace` | `kubeflow` | K8s namespace for tests |
| `--verbose` | false | Show detailed output |
| `--timeout` | `10000` | Request timeout in ms |

## When to Run

These tests are designed for:

1. **PR validation** - After server-side changes (ESM conversion, dependency upgrades)
2. **Pre-release** - Before cutting a release
3. **Debugging** - When investigating server issues

### Relevant PRs

- K8s client upgrades (`@kubernetes/client-node`)
- ESM/module system changes
- Express/middleware upgrades
- Server handler changes

## CI Integration

These tests require a running cluster and are not part of the standard unit test suite. To run in CI:

```yaml
- name: Setup Kind cluster
  uses: helm/kind-action@v1

- name: Deploy KFP
  run: make -C backend kind-cluster-agnostic

- name: Server integration tests
  run: |
    cd test/server-integration-test
    node server-test.js --url http://127.0.0.1:3000
    node k8s-integration-test.js --url http://127.0.0.1:3000
```

## Exit Codes

- `0` - All tests passed
- `1` - One or more tests failed

## Example Output

```
╔══════════════════════════════════════════════════════════════╗
║     K8s Integration Test - PR #12756 Validation              ║
╚══════════════════════════════════════════════════════════════╝

Target: http://127.0.0.1:3000
Namespace: kubeflow

☸️  K8s Pod Operations
  Find a running pod in namespace... ✓ (57ms)

📜 Pod Logs (K8s Client readNamespacedPodLog)
  Request logs for ml-pipeline pod... ✓ (69ms)
  Request logs with container parameter... ✓ (53ms)

📊 Tensorboard Viewer (K8s Client CRUD)
  GET tensorboard status (before create)... ✓ (26ms)
  POST create tensorboard (K8s createNamespacedPod)... ✓ (4ms)
  DELETE tensorboard (K8s deleteNamespacedPod)... ✓ (22ms)

Results: 10 passed, 0 failed (10 total)
```
