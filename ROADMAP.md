### KFP ROADMAP

The purpose of this KFP Roadmap document is to outline the planned themes and items for future development and improvements of Kubeflow Pipelines (KFP). It categorizes these efforts into several key areas: User Experience, User Documentation, Core Platform Stability, Extensibility, Integrations, and Quality of Life, QE, Performance & Scalability, Security & Compliance, and Community & Governance. The document also provides information on how to contribute to KFP.

## New Features & Enhancements

Q4 2025
- [ ] **Reusable components for common GenAI use cases**: As a GenAI practitioner, I want pre-built KFP components for common tasks like RAG and LLM compression to accelerate development.
- [ ] **Workspace Support**: As a user, I want to share data across components within a pipeline. \[[KEP](https://github.com/kubeflow/pipelines/blob/master/proposals/11875-pipeline-workspace/README.md)\]
- [ ] **KFP Local Mode Parity with Remote KFP**: As a user, I want the same feature set available to me in KFP local mode as KFP remote. \[[Epic](https://github.com/kubeflow/pipelines/issues/11793)\]
- [ ] **Support for External Metadata Providers**: Allow KFP to support logging metadata to external providers such as MLFlow, WandB, etc..
- [ ] **Multi-User Standalone KFP Deployment**: As a KFP admin, I want to deploy KFP standalone in multi-user mode for both production and development.

Q1 2026
- [ ] **PVC as the Storage Backend (Alternative to S3)**: As a K8s admin, I would like to utilize K8s PVC and storage class abstractions for artifact storage.
- [ ] **Share Python Code between Components in a Pipeline Version**
- [ ] **KFP Events Support**: As a user, I want to trigger Pipeline runs, and potentially components based on events.

Q2 2026
- [ ] **100% Kubernetes Infra Abstraction in SDK**: As a data scientist, I don’t want to see Kubernetes in my SDK code.
- [ ] **KFP UI with KFP Local Mode**: As a user, I want to be able to spin up a KFP UI server which runs KFP locally, allowing for a low resource intensive and quick mode of KFP deployment locally.
- [ ] **Allow Deleting & Cleaning up Artifacts**
- [ ] **Multi-Cluster Support**
- [ ] **KFP Pipeline and Runs Provenance**: As a KFP user, I want to be able to tell who uploaded a pipeline, pipeline version, who scheduled a workflow job, and who started a pipeline run. If the job is triggered by a trigger (separate roadmap item), it should specify that as well.

Q3 2026
- [ ] **Ability for Human-in-the-Loop Usage within Pipelines**: Allow for a component/task to be paused or continued via user controls in the UI or elsewhere.
- [ ] **Debug Mode for Components**: Decorate a component and, in the context of a broader pipeline, when that component launches (or if/when it fails), it pauses and waits for input. You can then exec into it (from your local machine or a KFP-provisioned notebook) and debug it interactively. If you mount a RWM PVC to the component and a notebook, you can even use the notebook to interactively author/execute logic on the component. When you're done debugging, through user input set the component to complete and let the DAG progress.


## User Documentation

Q4 2025
- [ ] **KFP repo should be a source of truth for all KFP docs**: As a new KFP user, I want quick and easy ways to get started with KFP for common use cases.

Q1 2026
- [ ] **Robust KFP Quickstarts**

## Core Platform Stability

Q3 2025
- [ ] **MLMD Removal:** Remove KFP’s dependence on MLMD by managing Artifacts and Lineage information via KFP. \[[KEP](https://github.com/kubeflow/pipelines/pull/12147)\]
- [ ] **Standalone TLS supported Deployment in KFP**: Users should be able to deploy a standalone KFP that is TLS enabled

Q4 2025
- [ ] **Argo WF support**: Latest KFP should always support 2 most recent Argo releases
- [ ] **React 19 Upgrade**: Upgrade KFP UI to React 19
- [ ] **Bump Python support to 3.10**: Minimum python support should be 3.10 (3.9 EOL is 2025-10)
- [ ] **Support PostgreSQL**
- [ ] **Support Mysql 9**
- [ ] **Standalone Driver:** As a KFP user, I want to reduce the number of pods that execute per pipeline to just the user runtime pods.\[[KEP](https://github.com/kubeflow/pipelines/pull/12023)\]

Q1 2026
- [ ] **Improved KFP configuration handling**: KFP configuration handling is messy, this effort aims to consolidate configuration sources and reduce config change overhead.
- [ ] **KFP backend should not need to depend on the SDK**: As a user I find it burdensome that each executor has to pip install KFP sdk, or provision it in the executor image.

## Extensibility, Integrations, and QoL

Q4 2025
- [ ] **Model Registry Component Integration**: As a user, I want KFP to integrate seamlessly with model registries for managing and deploying models
- [ ] **Training Operator Integration**: As a user, I want to be able to easily create KFP distributed training jobs from my pipelines.

Q1 2026
- [ ] **KServe Component Integration**: As a user, I want KFP to integrate with KServe for deploying and serving machine learning models.
- [ ] **Consolidate kfp & kfp-kubernetes python packages**: As a developer, I want a simplified Python package structure for KFP to reduce complexity.

## QE, Performance & Scalability

Q1 2026
- [ ] **Improve KFP Testing**: As a developer, I want comprehensive and granular testing for KFP to ensure stability and reliability.

## Security & Compliance

Q4 2025
- [ ] **0 Critical, 0 High-level CVEs from KFP image scans**: As a security conscious user, I want KFP official images to be free of critical and high-level CVEs.
- [ ] **0 Critical, 0 High-level SNYK & GitHub code scans from KFP Frontend**: As a security conscious user, I want KFP frontend code to be free of critical and high-level security vulnerabilities.
- [ ] **Auth Support for standalone deployments**: As an administrator, I want to secure standalone KFP deployments with robust authentication mechanisms.

## Community & Governance

Ongoing:

- [ ] **Have at least three maintainers be able to perform KFP releases**: As a community member, I want to see enough maintainers capable of performing KFP releases to ensure continuity.
- [ ] **3 KFP Blogs by Q4 2025**: As a community member, I want to see more informative content about KFP development and usage.
- [ ] **KFP User tutorial videos on YouTube**: As a KFP user, I want to have visual guides and tutorials available on YouTube.
- [ ] **KFP Admin tutorial videos on YouTube**: As a KFP administrator, I want to have visual guides and tutorials available on YouTube for deployment and management.
- [ ] **A curated list of "good first issues" that can be easily found**: As a new contributor, I want easy access to introductory tasks to help me get started with contributing to KFP.
- [ ] **At least four maintainers in total come from different organizations and/or stakeholders.**: As a community observer, the health of the KFP community is in part determined by the diversity and (aptly) sized governance.

## How to Contribute

* Propose new ideas: Open an issue!
* Join Weekly KFP community calls.
* PRs are welcome: see [CONTRIBUTING.md](http://contributing.md).
