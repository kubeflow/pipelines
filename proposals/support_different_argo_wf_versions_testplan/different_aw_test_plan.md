# Test Plan to verify multiple supported argo versions

## Compatibility Matrix Tests
Based on the compatability matrix as defined in #Test Environments

### Test Environments
| Environment | Argo Version  | KFP Version | Purpose                   |
|-------------|---------------|-------------|---------------------------|
| Env-1       | Latest(3.7.x) | None        | Future Upgrade            |
| Env-2       | 3.6.x         | Current     | N version compatibility   |
| Env-3       | 3.5.y         | Previous    | N-1 version compatibility |


### 1. Current Version (N) Compatibility

| Test Case ID          | TC-CM-001                                                                                                                                                                                                                                                                        |
|-----------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Validate compatibility with current supported Argo version                                                                                                                                                                                                                       |
| **Test Steps**        | <ol><li> Install current supported Argo version (e.g., 3.6.x) </li><li>Configure KFP with this Argo version</li><li>Execute pipeline run</li><li>Verify all features work correctly (Pipeline Run, and Scheduled Pipeline Runs works)</li><li>Document any limitations</li></ol> |
| **Expected Results**  | <ul><li>Full compatibility with current version</li><li> All pipeline features operational</li><li> No breaking changes or issues</li><li> Performance within acceptable range </li>                                                                                             |

### 2. Previous Version (N-1) Compatibility

| Test Case ID          | TC-CM-002                                                                                                                                                                                                                                   |
|-----------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Validate compatibility with previous supported Argo version                                                                                                                                                                                 |
| **Test Steps**        | <ol><li> Install previous supported Argo version (e.g., 3.5.y) </li><li>Configure KFP with this Argo version</li><li>Execute pipeline run</li><li>Document compatibility differences</li><li>Verify core functionality maintained</li></ol> |
| **Expected Results**  | <ul><li>Core functionality works with N-1 version</li><li> Any limitations clearly documented</li><li> No critical failures or data loss</li><li> Upgrade path available  </li>                                                             |

### 2.1 Z-Stream Version Compatibility

| Test Case ID          | TC-CM-002a                                                                                                                                                                                                                                                                                                                                                                  |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Validate compatibility with z-stream (patch) versions of Argo                                                                                                                                                                                                                                                                                                               |
| **Test Steps**        | <ol><li> Test current KFP with multiple z-stream versions of same minor Argo release<ul><li>Example: Test KFP with Argo v3.4.16, v3.4.17, v3.4.18</li></ul></li><li>Execute standard pipeline test suite for each z-stream version</li><li>Document any breaking changes in patch versions</li><li>Verify backward and forward compatibility within minor version</li></ol> |
| **Expected Results**  | <ul><li>Z-stream versions maintain compatibility </li><li> No breaking changes in patch releases</li><li> Smooth operation across patch versions</li><li> Clear documentation of any exceptions </li>                                                                                                                                                                       |

### 3. Version Matrix Validation

| Test Case ID          | TC-CM-003                                                                                                                                                                                                                                                                                           |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Test Case Summary** | Systematically validate compatibility matrix                                                                                                                                                                                                                                                        |
| **Test Steps**        | <ol><li> For each version in compatibility matrix: <ul><li> Deploy specific Argo version </li><li> Configure KFP<br/> </li><li> Execute standard test suite </li><li> Document results and issues</li><li>Update compatibility matrix</li></ul></li><li>Identify unsupported combinations</li></ol> |
| **Expected Results**  | <ul><li>Compatibility matrix accurately reflects reality</li><li> All supported versions documented</li><li> Unsupported combinations identified</li><li> Clear guidance for version selection  </li>                                                                                               |

## Regression Testing
Verify that for each PR, integration tests and KFP Sample Pipeline tests are successful