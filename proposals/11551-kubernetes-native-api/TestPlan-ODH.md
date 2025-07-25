## Feature Summary
Please refer [this](https://github.com/kubeflow/pipelines/blob/master/proposals/11551-kubernetes-native-api/README.md) for more details

## Test Sections

### Cluster Config (with FIPS)
| **Test Case**                                                                     | **Test Steps**                                        | **Expected Result**                                                                                 |
|-----------------------------------------------------------------------------------|-------------------------------------------------------|-----------------------------------------------------------------------------------------------------|
| (K8s mode) Create pipeline and a pipeline version using CRs                       | Match the pipeline and pipeline version (DB vs K8s)   | Pipeline and Pipeline version creation should be successful                                         |
| (K8s mode) Create pipeline and 2 pipeline versions with same spec using CRs       | Match the pipeline and pipeline versions (DB vs K8s)  | Pipeline and Pipeline versions creation should be successful                                        |
| (K8s mode) Create pipeline and 2 pipeline versions with different specs using CRs | Match the pipelines and pipeline versions (DB vs K8s) | Pipeline and Pipeline versions creation should be successful                                        |
| (K8s mode) Create pipeline run                                                    | Get Run details and validate                          | Pipeline run should succeed                                                                         |
| (K8s mode) Create an experiment and a pipeline run                                | Get Experiment and Run details and validate           | Pipeline run should be correctly associated to the created experiment and run should be successful  |

### Cluster Config (in ODH)
| **Test Case**                                                                     | **Test Steps**                                        | **Expected Result**                                                                                |
|-----------------------------------------------------------------------------------|-------------------------------------------------------|----------------------------------------------------------------------------------------------------|
| (K8s mode) Create pipeline and a pipeline version using CRs                       | Match the pipeline and pipeline version (DB vs K8s)   | Pipeline and Pipeline version creation should be successful                                        |
| (K8s mode) Create pipeline and 2 pipeline versions with same spec using CRs       | Match the pipeline and pipeline versions (DB vs K8s)  | Pipeline and Pipeline versions creation should be successful                                       |
| (K8s mode) Create pipeline and 2 pipeline versions with different specs using CRs | Match the pipelines and pipeline versions (DB vs K8s) | Pipeline and Pipeline versions creation should be successful                                       |
| (K8s mode) Create pipeline run                                                    | Get Run details and validate                          | Pipeline run should succeed                                                                        |
| (K8s mode) Create an experiment and a pipeline run                                | Get Experiment and Run details and validate           | Pipeline run should be correctly associated to the created experiment and run should be successful |

### Cluster Config (Disconnected RHOAI)
| **Test Case**                                                                     | **Test Steps**                                        | **Expected Result**                                                                                |
|-----------------------------------------------------------------------------------|-------------------------------------------------------|----------------------------------------------------------------------------------------------------|
| (K8s mode) Create pipeline and a pipeline version using CRs                       | Match the pipeline and pipeline version (DB vs K8s)   | Pipeline and Pipeline version creation should be successful                                        |
| (K8s mode) Create pipeline and 2 pipeline versions with same spec using CRs       | Match the pipeline and pipeline versions (DB vs K8s)  | Pipeline and Pipeline versions creation should be successful                                       |
| (K8s mode) Create pipeline and 2 pipeline versions with different specs using CRs | Match the pipelines and pipeline versions (DB vs K8s) | Pipeline and Pipeline versions creation should be successful                                       |
| (K8s mode) Create pipeline run                                                    | Get Run details and validate                          | Pipeline run should succeed                                                                        |
| (K8s mode) Create an experiment and a pipeline run                                | Get Experiment and Run details and validate           | Pipeline run should be correctly associated to the created experiment and run should be successful |
