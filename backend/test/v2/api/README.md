# KFP v2 API Test Framework

This directory contains the comprehensive integration test suite for Kubeflow Pipelines (KFP) v2 APIs. The test framework is built using **Ginkgo** and **Gomega** testing libraries and provides end-to-end testing for all API services.

## Table of Contents

1. [Test Framework Architecture](#test-framework-architecture)
2. [Test Coverage](#test-coverage)
3. [Test Creation Process](#test-creation-process)
4. [Directory Structure](#directory-structure)
5. [Running Tests](#running-tests)
6. [Contributing](#contributing)

## Test Framework Architecture

### Ginkgo & Gomega Integration

The test framework leverages:

- **Ginkgo v2**: BDD-style testing framework for organizing and running tests
- **Gomega**: Matcher library for assertions and expectations

#### Key Framework Components

**1. Test Suite Setup (`integration_suite_test.go`)**
- Global `BeforeSuite()`: Initializes API clients, creates directories, sets up Kubernetes client
- Global `BeforeEach()`: Sets up test-specific variables and common test data
- Global `AfterEach()`: Performs cleanup operations (deletes pipelines, runs, experiments)
- `ReportAfterEach()`: Captures logs and test reports on failure

**2. Configuration & Flags (`flags.go`)**
- Configurable test parameters via command-line flags
- Environment-specific settings (namespace, timeouts, debug mode)
- Test execution modes (dev mode, integration tests, proxy tests)

```go
// Example flag usage
var namespace = flag.String("namespace", "kubeflow", "The namespace ml pipeline deployed to")
var isDebugMode = flag.Bool("isDebugMode", false, "Whether to enable debug mode")
```

### Logging System (`logger/`)

**Custom Logger Implementation:**
- Integrates with Ginkgo's `GinkgoWriter` for test output
- Provides structured logging with format support
- Captures logs for failed test analysis

```go
// Usage example
logger.Log("Uploading pipeline file %s", pipelineFile)
logger.Log("Created Pipeline Run with id: %s", runID)
```

### Utility Classes (`utils/`)

**Core Utility Files:**
- `config_utils.go`: Client configuration and connection setup
- `pipeline_utils.go`: Pipeline CRUD operations and validation
- `pipeline_run_utils.go`: Pipeline run lifecycle management
- `experiment_utils.go`: Experiment operations
- `kubernetes_utils.go`: Kubernetes cluster interactions
- `file_utils.go`: File system operations for test data
- `test_utils.go`: Common test helpers and utilities

**Example Utility Usage:**
```go
// Pipeline operations
uploadedPipeline := utils.CreatePipeline(client, pipelineFile)
utils.DeletePipeline(client, pipelineID)

// Run operations
utils.WaitForRunToBeInState(runClient, runID, []State{RUNNING})
utils.TerminatePipelineRun(runClient, runID)
```

### Custom Matchers (`matcher/`)

**Specialized Assertion Matchers:**
- `custom_matcher.go`: Generic map comparison utilities
- `pipeline_matcher.go`: Pipeline-specific validation matchers

```go
// Custom matcher usage
MatchMaps(actualMap, expectedMap, "pipeline parameters")
```

### Constants & Configuration (`constants/`)

**Test Classification:**
- `test_type.go`: Test categories (Smoke, CriticalOnly, FullRegression)

**Example Test Labeling:**
```go
var _ = Describe("Pipeline API Tests", Label("Positive", "Pipeline", FullRegression), func() {
    // Test implementation
})
```

### Reports Generation

**Automated Report Generation:**
- **JUnit XML Reports**: `reports/api.xml` for CI/CD integration
- **JSON Reports**: `reports/api.json` for detailed analysis
- **Test Logs**: Individual test logs in `logs/` directory
- **Pod Logs**: Captured on test failures for debugging

```go
// Report configuration
reporterConfig.JUnitReport = filepath.Join(testReportDirectory, junitReportFilename)
reporterConfig.JSONReport = filepath.Join(testReportDirectory, jsonReportFilename)
```

## Test Coverage

### API Service Test Distribution

Each API service has dedicated test files with comprehensive endpoint coverage:

| API Service             | Test File                            | Primary Focus                        | Test Count             |
|-------------------------|--------------------------------------|--------------------------------------|------------------------|
| **Pipeline API**        | `pipeline_api_test.go`               | Pipeline CRUD, versioning, listing   | 30+ test scenarios     |
| **Pipeline Upload API** | `pipeline_upload_api_test.go`        | File upload, validation, versioning  | 15+ test scenarios     |
| **Pipeline Run API**    | `pipeline_run_api_test.go`           | Run lifecycle, state management      | 45+ test scenarios     |
| **Experiment API**      | `experiment_api_test.go`             | Experiment management, association   | 25+ test scenarios     |
| **Recurring Run API**   | `pipeline_recurring_run_api_test.go` | Scheduled runs, cron jobs            | 20+ test scenarios     |
| **Report API**          | `report_api_test.go`                 | Workflow reporting, metrics          | 10+ test scenarios     |
| **E2E Pipeline**        | `pipeline_e2e_test.go`               | End-to-end pipeline execution        | 5+ comprehensive flows |

### Endpoint Coverage Details

**Pipeline API Coverage:**
- ✅ List operations (pagination, sorting, filtering)
- ✅ CRUD operations (create, read, update, delete)
- ✅ Version management
- ✅ Namespace isolation
- ✅ Error handling and validation

**Pipeline Run API Coverage:**
- ✅ Run creation and execution
- ✅ State transitions (PENDING → RUNNING → SUCCESS/FAILED)
- ✅ Run termination and cancellation
- ✅ Archive/unarchive operations
- ✅ Experiment association
- ✅ Parameter handling and validation

**Experiment API Coverage:**
- ✅ Experiment lifecycle management
- ✅ Run association and disassociation
- ✅ Archive/unarchive workflows
- ✅ Filtering and search operations

### Test Categorization

**By Test Type:**
- **Smoke Tests**: Critical path validation
- **Regression Tests**: Comprehensive feature coverage
- **Integration Tests**: Cross-service interaction validation

**By Severity:**
- **S1 (Critical)**: Core functionality, blocking issues
- **S2 (Medium)**: Important features, moderate impact
- **S3 (Trivial)**: Edge cases, minor functionality

## Test Creation Process

### 1. Test Structure Pattern

All tests follow a consistent BDD structure using Ginkgo's descriptive syntax:

```go
var _ = Describe("API Feature Description >", Label("Positive", "ServiceName", FullRegression), func() {
    Context("Specific Scenario Group >", func() {
        It("Should perform specific action", func() {
            // Test implementation
        })
    })
})
```

### 2. Test Implementation Lifecycle

**Setup Phase:**
```go
BeforeEach(func() {
    logger.Log("################### Setup before each test #####################")
    // Initialize test-specific variables
    // Create test data
    // Setup API clients
})
```

**Test Execution:**
```go
It("Should validate specific behavior", func() {
    // 1. Arrange: Setup test data
    logger.Log("Starting test: %s", testDescription)
    
    // 2. Act: Execute the operation
    result := performAPICall(testData)
    
    // 3. Assert: Validate results
    Expect(result).To(Equal(expectedResult))
    Expect(result.Status).To(Equal("SUCCESS"))
})
```

**Cleanup Phase:**
```go
AfterEach(func() {
    logger.Log("################### Global Cleanup after each test #####################")
    // Clean up created resources
    // Delete test data
    // Reset state
})
```

### 3. Logging Best Practices

**Structured Logging:**
```go
// Test phase logging
logger.Log("################### Setup Phase #####################")
logger.Log("Creating pipeline with name: %s", pipelineName)
logger.Log("Pipeline created successfully with ID: %s", pipelineID)
logger.Log("################### Cleanup Phase #####################")
```

**Error Context Logging:**
```go
logger.Log("Test failed... Capturing pod logs")
podLogs := utils.ReadContainerLogs(k8Client, namespace, "pipeline-api-server", &testStartTimeUTC)
AddReportEntry("Pod Log", podLogs)
```

### 4. Resource Management

**Automatic Cleanup:**
- All created resources are tracked in global arrays
- Cleanup happens in `AfterEach()` regardless of test outcome
- Resources include: pipelines, pipeline runs, experiments

```go
// Resource tracking
createdPipelines = []*upload_model.V2beta1Pipeline{}
createdRunIds = make([]string, 0)
createdExperimentIds = make([]string, 0)

// Automatic cleanup
for _, pipeline := range createdPipelines {
    utils.DeletePipeline(pipelineClient, pipeline.PipelineID)
}
```

### 5. Error Handling & Validation

**Assertion Patterns:**
```go
// Standard assertions
Expect(err).NotTo(HaveOccurred(), "Operation should succeed")
Expect(response.Status).To(Equal(expectedStatus))
Expect(len(results)).To(BeNumerically(">=", 1))

// Custom matcher usage
MatchMaps(actualParameters, expectedParameters, "pipeline parameters")
```

**State Validation:**
```go
// Wait for expected states
utils.WaitForRunToBeInState(runClient, &runID, 
    []run_model.V2beta1RuntimeState{
        run_model.V2beta1RuntimeStateRUNNING,
    }, 
    nil)
```

## Directory Structure

```
backend/test/v2/api/
├── README.md                          # This documentation
├── integration_suite_test.go          # Main test suite setup
├── flags.go                          # Test configuration flags
├── constants/                        # Test constants and enums
│   ├── severity.go                   # Test severity levels
│   └── test_type.go                  # Test type categories
├── logger/                           # Logging utilities
│   └── logger.go                     # Custom logger implementation
├── matcher/                          # Custom assertion matchers
│   ├── custom_matcher.go             # Generic map matchers
│   └── pipeline_matcher.go           # Pipeline-specific matchers
├── utils/                            # Utility functions
│   ├── config_utils.go               # Client configuration
│   ├── experiment_utils.go           # Experiment operations
│   ├── file_utils.go                 # File system operations
│   ├── kubernetes_utils.go           # K8s cluster operations
│   ├── pipeline_run_utils.go         # Pipeline run management
│   ├── pipeline_utils.go             # Pipeline operations
│   ├── pipeline_version_utils.go     # Version management
│   └── test_utils.go                 # Common test helpers
├── logs/                             # Test execution logs
├── reports/                          # Generated test reports
│   ├── api.xml                       # JUnit XML report
│   └── api.json                      # JSON test report
├── *_api_test.go                     # Individual API test files
├── pipeline_e2e_test.go              # End-to-end tests
└── argo_workflow_converter.go        # Workflow conversion utilities
```

## Running Tests

### Basic Test Execution

```bash
# Run all API tests
go test -v ./backend/test/v2/api/

# Run with specific flags
go test -v ./backend/test/v2/api/ \
  -namespace=kubeflow \
  -isDebugMode=true \
  -runIntegrationTests=true
```

### Test Filtering

```bash
# Run only smoke tests
ginkgo -v --label-filter="Smoke" ./backend/test/v2/api/

# Run critical severity tests
ginkgo -v --label-filter="S1" ./backend/test/v2/api/

# Run pipeline-specific tests
ginkgo -v --label-filter="Pipeline" ./backend/test/v2/api/
```

### Configuration Options

| Flag | Default | Description |
|------|---------|-------------|
| `namespace` | `kubeflow` | Target Kubernetes namespace |
| `isDebugMode` | `false` | Enable detailed debug logging |
| `runIntegrationTests` | `true` | Execute integration tests |
| `isKubeflowMode` | `false` | Run in full Kubeflow mode |
| `cacheEnabled` | `true` | Enable pipeline caching |
| `podLogLimit` | `50000000` | Limit for captured pod logs |

## Contributing

### Adding New Tests

1. **Create test file**: Follow naming convention `*_api_test.go`
2. **Import required packages**: Include framework imports and utilities
3. **Define test structure**: Use BDD patterns with proper labeling
4. **Implement test logic**: Follow the setup → act → assert → cleanup pattern
5. **Add proper logging**: Use structured logging throughout
6. **Update documentation**: Document new test coverage

### Test Development Guidelines

- **Use descriptive test names** that clearly indicate the test purpose
- **Implement proper cleanup** to avoid test pollution
- **Add appropriate labels** for test categorization and filtering
- **Include comprehensive logging** for debugging failed tests
- **Follow consistent code patterns** established in existing tests
- **Validate all assumptions** with explicit assertions

### Code Review Checklist

- [ ] Tests follow BDD structure and naming conventions
- [ ] Proper resource cleanup is implemented
- [ ] Comprehensive logging is included
- [ ] Test labels are correctly applied
- [ ] Error cases are handled appropriately
- [ ] Documentation is updated if needed 