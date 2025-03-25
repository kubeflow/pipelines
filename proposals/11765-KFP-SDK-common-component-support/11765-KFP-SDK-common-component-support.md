# Python SDK Common Components for Kubeflow Pipelines
#### A Kubeflow Enhancement Proposal (KEP)

## Introduction

While KFP excels as a pipeline execution platform, the current approach to providing reusable components, particularly through the [`pipelines/components`](../../components) directory, falls short of delivering a smooth and efficient user experience for Machine Learning Engineers (MLEs) on KFP.

This proposal focuses on a critical first step towards improving KFP component usability: **replacing the existing [`pipelines/components`](../../components) directory with a more robust and user-friendly SDK-centric approach.**  This involves embedding core, widely applicable components directly within the KFP SDK; these components should be organized logically and easily accessible to users. This shift aims to reduce the burden on MLEs of finding or building common components, allowing them to focus more on ML development and less on component wrangling.

## Motivation: Benefits of Python SDK-Centric Components

For MLE users:
* **Improved User Experience:**  Easier component discovery, import, and usage directly within the KFP SDK. A more Python-centric workflow reduces friction for MLEs; this also makes for a more powerful experience when used in the context of Python IDE support for Python docstrings, debugging, syntax highlighting and etc.
* **Faster Development Iteration:**  Ready-to-use Python-based SDK components can facilitate rapid prototyping and iteration, aligning with the iterative nature of ML development.
* **Better Integration with Kubeflow Ecosystem:**  Prioritizing integration with core Kubeflow projects ensures a cohesive and well-supported ecosystem for Kubeflow and KFP users.

For KFP Developers:
* **Enhanced Component Quality and Consistency:**  A curated set of SDK components allows for focused development, testing, and maintenance, leading to higher quality and more reliable components.
* **Simplified Component Management:**  Centralized component management within the SDK reduces fragmentation and maintenance overhead compared to the current [`pipelines/components`](../../components) directory.

## Problem Statement: Current Component Challenges

The current KFP component resources suffer from several issues:

* **Disorganized and Inconsistent [`pipelines/components`](../../components) Directory:** The existing directory structure is vendor-centric, making it difficult to discover components based on functionality. Components within this directory are often POC-level, outdated, and lack consistent quality, documentation, and usability. MLEs prefer to work with Python; however, many components in the [`pipelines/components`](../../components) directory rely on YAML or containerized components only, and they have little or no documentation.
* **Lack of Python SDK-Level Support for components:**  The KFP SDK currently focuses on component *creation* and *loading* but lacks a built-in support with readily available, high-quality components. This forces users to either build components from scratch, or go to the repo and navigate the fragmented [`pipelines/components`](../../components) directory.

These issues lead to inefficiencies, increased development time, and a less-than-ideal user experience; they ultimately get in the way of wider adoption and effective use of KFP for ML workflows.

## Proposal: SDK Support for Common Components

This proposal advocates for a fundamental shift in how KFP provides common component support: **integration directly into the KFP SDK.**  This involves:

* **Deprecate and Replacing the [`pipelines/components`](../../components) Directory:**
    * **Retire Outdated/POC Components:**  Identify and remove components in [`pipelines/components`](../../components) that are outdated, buggy, or lack sufficient quality.
    * **Migrate Useful Examples:**  Some components appear to be meant more as proof-of-concepts, than as production-ready tools; these are better kept in the [`samples`](../../samples/) directory. Relocate valuable, proof-of-concept components to the [`samples`](../../samples/) directory as illustrative examples.
    * **Discontinue Active Development:**  Cease further development and maintenance of the [`pipelines/components`](../../components) directory as the primary source of reusable components.

* **Develope SDK-Integrated Component Categories:**  Organize core, widely applicable components within the KFP SDK under logical functional categories.  This proposal suggests the following initial categories (subject to review), mirroring common ML workflow stages:

    * **`kfp.components.data`:**  Components for data ingestion, transformation, and validation. Conceptual examples:
        * `load_from_gcs(...)`: Load data from Google Cloud Storage.
        * `split_dataset(...)`: Split a dataset into training and validation sets.
        * `validate_schema(...)`: Validate data against a schema.

    * **`kfp.components.train`:** Components for model training and hyperparameter tuning.  Conceptual examples:
        * **`kubeflow_trainer(...)`:**  Integration with Kubeflow Training Operators
        * **`hyperparameter_tune(...)`:**  Integration with Katib for hyperparameter optimization.
        * `xgboost.train(...)`:  Component for XGBoost training.

    * **`kfp.components.serve`:** Components for model deployment and serving. Conceptual examples:
        * **`deploy_kserve(...)`:** Integrate with KServe for model serving on Kubernetes.
        * `register_model(...)`:  Register a model in a model registry.

    * **`kfp.components.validate`:** Components for model evaluation and monitoring. Conceptual examples:
        * `evaluate_classification_model(...)`:  Calculate standard evaluation metrics.
        * `data_drift_detection(...)`: Detect data drift using statistical methods.

Note that the conceptual examples here are for illustrative purposes, and are not meant to be the final say.

* **Implementation Details:**

    * **Python-Centric Design:** Components will be primarily Python function-based, leveraging the `@kfp.dsl.component` decorator for ease of use and local testing.
    * **Base Image Utilization:** Each Python SDK component will have an acceptible default publicly available base image (for example [pytorch/pytorch](https://hub.docker.com/r/pytorch/pytorch)).
    * **Python Dependencies** Python dependencies not in the base image will be the default list in the `packages_to_install` parameter.
    * **Clear Documentation:** Each SDK component will have comprehensive documentation, including:
        * Purpose and functionality
        * Input and output specifications
        * Dependency requirements (packages, environment variables, cluster CRDs, Secrets etc.)
        * Usage examples
        * Guidance on base image selection and other parameter options.
    * **Focus on Core Kubeflow Integrations:** Prioritize components that integrate seamlessly with core Kubeflow projects like Kubeflow Trainer, Katib, and KServe.
    * **Focus on User Value and Simple Design** Components that make it into the Python SDK should be there because they are widely useful in MLE work. Simplicity is a virtue. Since each new component brings complexity to the SDK, it must also bring a high level of value. It may be possible to include more narrow use case components in library extensions.

The core value-add of this proposal is to suggest the structure necessary to support a basic set of high quality, readily available SDK-level KFP components. This foundation should create more capability for ML engineers.

## Technical Changes and Implementation Steps

To implement this proposal, the following technical changes and steps are required:

1. **Define and Implement Top-Level `kfp.components` Structure:**  Create the proposed `kfp.components` namespace sub-modules (e.g., `data`, `train`, `serve`, `validate`) within the KFP SDK.
2. **Develop Core Component Set:**  Prioritize the development of a core set of commonly used components within each category, focusing initially on integrations with Kubeflow Trainer, Katib, and KServe.
3. **Migrate and Refactor Existing Components (Where Applicable):**  Evaluate components in [`pipelines/components`](../../components) for potential refactoring and migration to the SDK. Focus on components with broad applicability and high quality. Move components that serve more as good examples to the `samples` directory.
4. **Deprecate [`pipelines/components`](../../components) Directory:**  Formally announce the deprecation of the [`pipelines/components`](../../components) directory and guide users towards the new SDK-centric approach.
5. **Update Documentation and Examples:**  Thoroughly document the new `kfp.components` structure and provide comprehensive usage examples in the KFP documentation and samples.
6. **Establish Component Development Guidelines:**  Define clear guidelines for developing and contributing SDK components to maintain quality and consistency.

## Follow-up Ideas

While the details are out of scope of this proposal, two natural follow-up ideas are:
* **SDK Extensions for components:** SDK extensions may make sense for component sets that support specific environments (like specific clouds), that are still widely useful, but only in certain contexts.
* **Component Hub:** A component hub would be a place where the KFP community can upload components to share. ML users are familiar with this pattern, for example with [PyTorch Hub](https://pytorch.org/hub/), and the ability to load a model through the PyTorch SDK with `torch.hub.load(<model_name>)`. It would be very intuitive for ML engineers to use the same functionality with a "KFP Hub" and SDK tools like `component = kfp.hub.load(<component_name>)`.  

This proposal does not seek to flesh out these ideas, but rather to focus on getting the core SDK components from idea to reality. Let's get the core in place first. Presumably, both extensions and a hub would follow the component precedents for structure and quality that are set in this proposal.

## Conclusion

This technical proposal for SDK-centric components represents a crucial step towards significantly improving the KFP user experience. By replacing the fragmented and less user-friendly [`pipelines/components`](../../components) directory with a well-structured and integrated component set within the KFP SDK, we can help MLEs to build and deploy ML pipelines more efficiently. This approach will pave the way for a more robust, user-friendly, and widely adopted Kubeflow Pipelines platform, aligning with the broader vision of simplifying and streamlining MLOps workflows.