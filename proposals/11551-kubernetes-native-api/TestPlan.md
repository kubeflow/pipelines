## Feature Summary
Please refer [this](https://github.com/kubeflow/pipelines/blob/master/proposals/11551-kubernetes-native-api/README.md) for more details

## Test Sections
### Migration of existing pipelines
#### With the migration script
| **Test Case**                                                                                                                        | **Test Steps**                                                                                                                       | **Expected Result**                                                                                          |
|--------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------|
| (KFP DB mode) Create a pipeline and a pipeline version and run migrate script                                                        |                                                                                                                                      | Pipeline and pipeline version should match exactly except for ID?                                            |
| (KFP DB mode) Create a pipeline and 2 pipeline versions with same spec and run migrate script                                        |                                                                                                                                      | Pipeline and pipeline versions should match exactly except for ID?                                           |
| (KFP DB mode) Create a pipeline and 2 pipeline versions with different specs and run migrate script                                  |                                                                                                                                      | Pipeline and pipeline versions should match exactly except for ID?                                           |
| (KFP DB mode) Create 2 pipelines and 1 pipeline version per pipeline spec and run migrate script                                     |                                                                                                                                      | Pipelines and pipeline versions should match exactly except for IDs?                                         |
| (KFP DB mode) Create 2 pipeline and 2 pipeline versions with different specs and run migrate script                                  |                                                                                                                                      | Pipelines and pipeline versions should match exactly except for IDs?                                         |
| (KFP DB mode) Create all pipelines and 1 pipeline version per pipeline spec under "data/pipeline_files/valid" and run migrate script | Run [Pipeline Upload API Tests](https://github.com/nsingla/kfp_pipelines/tree/pipeline_run_tests) without clean up and run migration | Confirm that the migration script does not error out and is able to migrate all pipelines and their versions |
| (K8s Mode) Create a pipeline in DB mode and switch to k8s mode and try creating the same pipeline without migration                  |                                                                                                                                      | Should disallow duplicate pipeline which hasn't been migrated?                                               |
| (K8s Mode) Create a pipeline run in an experiment in DB mode and while the run is in progress switch to k8s mode                     |                                                                                                                                      | The run continue after migration                                                                             |
| (K8s Mode) Create a pipeline recurring run in an experiment in DB mode and switch to k8s mode                                        |                                                                                                                                      | The run should still exist with the same cron settings                                                       |
| --------------After migration--------------------                                                                                    |
| (K8s Mode) Try creating an existing pipeline and pipeline verison                                                                    |                                                                                                                                      | Pipeline should not get created with a duplicate name                                                        |
| (K8s Mode) Run an existing pipeline and get run details and match                                                                    |                                                                                                                                      |                                                                                                              |
| (K8s Mode) Run all migrated pipeline and match the run details                                                                       |                                                                                                                                      |                                                                                                              |
| (K8s Mode) Create an experiment and a run                                                                                            |                                                                                                                                      |                                                                                                              |

#### Without migration script
| **Test Case**                                                                                                                                                                  | **Test Steps**                                        | **Expected Result** |
|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------|---------------------|
| (KFP DB mode) Create pipeline and a pipeline version and re-create the pipeline & version using CRs                                                                            | Match the pipeline and pipeline version (DB vs K8s)   |                     |
| (KFP DB mode) Create pipeline and 2 pipeline versions with same spec and and re-create the pipeline & version using CRs                                                        | Match the pipeline and pipeline versions (DB vs K8s)  |                     |
| (KFP DB mode) Create pipeline and 2 pipeline versions with different specs and and re-create the pipeline & version using CRs                                                  | Match the pipelines and pipeline versions (DB vs K8s) |                     |
| (KFP DB mode) Create pipeline and a pipeline version and and re-create the pipeline & version using CRs but with different **pipelineName** than the one in DB in pipelineSpec |                                                       | ???                 |
| (K8s mode) Create pipeline run for a newly migrated pipeline associated with the default experiment                                                                            | Get Run details and validate                          |                     |
| (K8s mode) Create an experiment and a pipeline run associating it with a migrated pipeline                                                                                     | Get Experiment and Run details and validate           |                     |
| (K8s mode) Rerun a pipeline run associated with an existing experiment and a migrated pipeline version                                                                         | Get Experiment and Run details and validate           |                     |


### New Pipelines
| **Test Case**                                                                                | **Test Steps**                                        | **Expected Result**                                                                                                        |
|----------------------------------------------------------------------------------------------|-------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------|
| (K8s mode) Create pipeline (via API) and a pipeline version using CRs                        | Match the pipeline and pipeline version (DB vs K8s)   |                                                                                                                            |
| (K8s mode) Create pipeline (via API) and 2 pipeline versions with same spec using CRs        | Match the pipeline and pipeline versions (DB vs K8s)  |                                                                                                                            |
| (K8s mode) Create pipeline (via API) and 2 pipeline versions with different specs using CRs  | Match the pipelines and pipeline versions (DB vs K8s) |                                                                                                                            |
| (K8s mode) Create pipeline run (via API)                                                     | Get Run details and validate                          |                                                                                                                            |
| (K8s mode) Create an experiment and a pipeline run  (via API)                                | Get Experiment and Run details and validate           |                                                                                                                            |
| (K8s mode) One you have saved few pipelines in the K8s storage, run the migration script     |                                                       | What happens to the pipelines with the same name already existing in the K8s storage, will the migration script error out? |
| (K8s mode) Create pipeline (via UI) and a pipeline version using CRs                         | Match the pipeline and pipeline version (DB vs K8s)   |                                                                                                                            |
| (K8s mode) Create pipeline (via UI) and 2 pipeline versions with same spec using CRs         | Match the pipeline and pipeline versions (DB vs K8s)  |                                                                                                                            |
| (K8s mode) Create pipeline (via UI) and 2 pipeline versions with different specs using CRs   | Match the pipelines and pipeline versions (DB vs K8s) |                                                                                                                            |
| (K8s mode) Create pipeline run (via UI)                                                      | Get Run details and validate                          |                                                                                                                            |
| (K8s mode) Create an experiment and a pipeline run  (via UI)                                 | Get Experiment and Run details and validate           |                                                                                                                            |

### Custom K8s storage (with custom role binding)
Should we care about this? from migration script POV?

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
