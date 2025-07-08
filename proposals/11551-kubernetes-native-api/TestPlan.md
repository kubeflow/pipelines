## Feature Summary
Please refer [this](https://github.com/kubeflow/pipelines/blob/master/proposals/11551-kubernetes-native-api/README.md) for more details

## Test Sections
### Migration of existing pipelines
#### With the migration script
| **Test Case**                                                                                                                        | **Test Steps** | **Expected Result** |
|--------------------------------------------------------------------------------------------------------------------------------------|----------------|---------------------|
| (KFP DB mode) Create pipeline and a pipeline version and run migrate script                                                          |                |                     |
| (KFP DB mode) Create pipeline and 2 pipeline versions with same spec and run migrate script                                          |                |                     |
| (KFP DB mode) Create pipeline and 2 pipeline versions with different specs and run migrate script                                    |                |                     |
| (KFP DB mode) Create 2 pipelines and 1 pipeline version per pipeline spec and run migrate script                                     |                |                     |
| (KFP DB mode) Create 2 pipeline and 2 pipeline versions with different specs and run migrate script                                  |                |                     |
| (KFP DB mode) Create all pipelines and 1 pipeline version per pipeline spec under "data/pipeline_files/valid" and run migrate script |                |                     |
| (K8s Mode) Create a pipeline in DB mode and switch to k8s mode and try creating the same pipeline without migration                  |                |                     |
| --------------After migration--------------------                                                                                    |
| (K8s Mode) Try creating an existing pipeline and pipeline verison                                                                    |                |                     |
| (K8s Mode) Run an existing pipeline and get run details and match                                                                    |                |                     |
| (K8s Mode) Run all migrated pipeline and match the run details                                                                       |                |                     |
| (K8s Mode) Create an experiment and a run                                                                                            |                |                     |

#### Without migration script
| **Test Case**                                                                                                                                                                  | **Test Steps**                                        | **Expected Result** |
|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------|---------------------|
| (KFP DB mode) Create pipeline and a pipeline version and re-create the pipeline & version using CRs                                                                            | Match the pipeline and pipeline version (DB vs K8s)   |                     |
| (KFP DB mode) Create pipeline and 2 pipeline versions with same spec and and re-create the pipeline & version using CRs                                                        | Match the pipeline and pipeline versions (DB vs K8s)  |                     |
| (KFP DB mode) Create pipeline and 2 pipeline versions with different specs and and re-create the pipeline & version using CRs                                                  | Match the pipelines and pipeline versions (DB vs K8s) |                     |
| (KFP DB mode) Create pipeline and a pipeline version and and re-create the pipeline & version using CRs but with different **pipelineName** than the one in DB in pipelineSpec |                                                       | ???                 |


### New Pipelines

### Custom K8s storage (with custom role binding)
### Custom K8s storage (with FIPS)
| **Test Case**                                                                     | **Test Steps**                                        | **Expected Result** |
|-----------------------------------------------------------------------------------|-------------------------------------------------------|---------------------|
| (K8s mode) Create pipeline and a pipeline version using CRs                       | Match the pipeline and pipeline version (DB vs K8s)   |                     |
| (K8s mode) Create pipeline and 2 pipeline versions with same spec using CRs       | Match the pipeline and pipeline versions (DB vs K8s)  |                     |
| (K8s mode) Create pipeline and 2 pipeline versions with different specs using CRs | Match the pipelines and pipeline versions (DB vs K8s) |                     |
| (K8s mode) Create pipeline run                                                    | Get Run details and validate                          |                     |
| (K8s mode) Create an experiment and a pipeline run                                | Get Experiment and Run details and validate           |                     |

### Custom K8s storage (with Kubeflow)
| **Test Case**                                                                     | **Test Steps**                                        | **Expected Result** |
|-----------------------------------------------------------------------------------|-------------------------------------------------------|---------------------|
| (K8s mode) Create pipeline and a pipeline version using CRs                       | Match the pipeline and pipeline version (DB vs K8s)   |                     |
| (K8s mode) Create pipeline and 2 pipeline versions with same spec using CRs       | Match the pipeline and pipeline versions (DB vs K8s)  |                     |
| (K8s mode) Create pipeline and 2 pipeline versions with different specs using CRs | Match the pipelines and pipeline versions (DB vs K8s) |                     |
| (K8s mode) Create pipeline run                                                    | Get Run details and validate                          |                     |
| (K8s mode) Create an experiment and a pipeline run                                | Get Experiment and Run details and validate           |                     |

### Custom K8s storage (in ODH)
| **Test Case**                                                                     | **Test Steps**                                        | **Expected Result** |
|-----------------------------------------------------------------------------------|-------------------------------------------------------|---------------------|
| (K8s mode) Create pipeline and a pipeline version using CRs                       | Match the pipeline and pipeline version (DB vs K8s)   |                     |
| (K8s mode) Create pipeline and 2 pipeline versions with same spec using CRs       | Match the pipeline and pipeline versions (DB vs K8s)  |                     |
| (K8s mode) Create pipeline and 2 pipeline versions with different specs using CRs | Match the pipelines and pipeline versions (DB vs K8s) |                     |
| (K8s mode) Create pipeline run                                                    | Get Run details and validate                          |                     |
| (K8s mode) Create an experiment and a pipeline run                                | Get Experiment and Run details and validate           |                     |
