# Official Component Expectations and Standards for Kubeflow Pipelines
#### A Kubeflow Enhancement Proposal (KEP)

## Introduction

This proposal addresses the critical need for clear, official standards for Kubeflow Pipelines (KFP) components. The current lack of defined expectations for component structure, quality, usability, and documentation has led to inconsistencies, user confusion, and increased development burden. **This proposal focuses on establishing and implementing official component expectations and standards to improve the overall KFP user experience and create a more robust and trustworthy component ecosystem.**

## Motivation: Benefits of Official Component Expectations and Standards

* **Improved Component Quality and Reliability:**  Standards will drive higher quality KFP official components, reducing bugs, inconsistencies, and user frustration.
* **Enhanced User Experience:**  Clear expectations and standards for components will make KFP easier to learn and use, leading to more efficient pipeline development.
* **Increased User Trust and Adoption:**  Standardized, reliable components will foster greater user trust in the official KFP component ecosystem and encourage wider adoption.
* **Streamlined Contribution Process:**  Clear guidelines will simplify the contribution process, encouraging more community contributions of high-quality components.
* **Reduced Support Burden:**  Better components and documentation will decrease support issues related to component setup, usage, and debugging.
* **Template for Enterprise Standards:**  KFP's official standards can serve as a template for enterprises developing their own internal component standards for MLOps governance.

## Problem Statement: Lack of Clear Component Standards

The absence of clear official standards for KFP components results in several significant challenges:

* **Inconsistent Component Quality:** Components within the current `pipelines/components` directory exhibit varying levels of structure, quality, usability, and documentation. These inconsistencies makes it difficult for users to rely on official components and necessitates significant adaptation and debugging effort.
* **User Confusion and Frustration:** Without clear expectations, users struggle to understand component setup, dependencies, and intended usage. This applies to both components that users build, as well as components they rely on from the community. The lack of clarity increases the learning curve for KFP and hinders efficient pipeline development. Components that are poorly built can also cause problems in runtime environments, and across enterprise ML organizations.
* **Contribution Barriers:**  Contributors lack clear guidelines on what constitutes a high-quality, acceptable component for the official KFP repository. This ambiguity discourages contributions and arguably has led to a proliferation of inconsistent and subpar components in the past.
* **Increased Support Burden:**  The lack of standards results in increased support requests from users struggling with poorly documented or unreliable components.

These issues collectively undermine the KFP user experience and hinder the growth of the KFP component ecosystem.

## Proposed Solution: Define and Implement Official Component Expectations and Standards

This proposal advocates for the formal definition, documentation, and enforcement of official component expectations and standards within the KFP project, while also setting an example for users wishing to implement their own components. This proposal involves:

* **Define Official Component Expectations:**  Establish a clear set of expectations for official KFP components. These expectations will serve as guidelines for both component users and contributors. These expectations will include:

    * **Standards for Ease of Setup and Use:** Components must be designed for straightforward setup and intuitive usage from an MLE perspective. This includes facilitating transitions across different environments (local, container, cloud, cluster) and supporting rapid iteration and experimentation.
    * **Standards for Dependency Clarity:**  All component dependencies (code, data, and environment) must be explicitly and clearly documented. This includes detailing default and optional dependencies, as well as environment-specific behaviors (e.g., dry-run flags, GPU requirements, and etc.).
    * **Standards for Minimized Container Burden:** Components should minimize the need for users to build, host, and maintain custom container images for common use cases.  Leverage KFP's purpose-built functionalities (like Python code injection and data passing) and encourage the use of pre-built publicly available base images.
    * **Standards for Broad Applicability and Openness:** Official components should address widely applicable ML use-cases and prioritize open-source, cloud-native solutions aligned with CNCF standards.
    * **Standards for Production-Level Robustness:** Official components must be robust and suitable for production environments. This includes expectations for well-written, tested code, logging support, inline documentation, and reliance on reputable, secure dependencies with generally acceptable licensing.

* **Create a Component Checklist:** Develop a detailed checklist to guide component developers and reviewers. This checklist will serve as a practical tool to ensure adherence to the defined expectations. Key areas covered by the checklist can include:

    * **Python Function Documentation and Accessibility:**  Ensure core Python function is documented and usable both within and outside KFP pipelines.
    * **Dependency Declarations:**  Explicitly list Python libraries, environment variables, and PyPI index/server requirements. Also, note minimum KFP (SDK or backend) version requirements.
    * **Environment Setup and Requirements:**  Detail cluster dependencies (CRD, ConfigMap, Secret, etc), and system resource expectations (GPU, memory, CPU).
    * **Usage Instructions Across Environments:** Provide clear instructions for setting up and running components in different environments (local, cluster, etc.), considering differing needs for authentication, interacting with outputs, testing, and etc.
    * **Input/Output and Artifact Documentation:**  Clearly document component inputs, outputs, metadata, artifacts, errors, and logging.
    * **Performance and Testing Information:**  Provide guidance on expected runtime, resource usage, and include tests with instructions for execution.
    * **Ownership:** Make it clear who is responsible for the individual component maintenance (if applicable). Have procedures for transfering ownership and for deprecating abandoned components.

* **Document and Promote Standards:**  Clearly document the defined component expectations and the component checklist in prominent locations, such as:

    * **Top-level `README.md` in the KFP repository.**
    * **Dedicated section in the KFP website documentation.**
    * **Component-specific `README.md` files and/or SDK-level docstrings.**

* **Enforcement and Review Process:**  Establish a review process (documented in KFP repository) to ensure that new component contributions and updates adhere to the defined standards and utilize the component checklist. This process could involve:

    * **Code reviews specifically focusing on adherence to standards.**
    * **Automated checks to verify documentation and checklist completion.**
    * **Community feedback and validation of component quality.**

## Implementation Steps

To implement this proposal, the following steps are necessary:

1. **Formalize Component Expectations:**  Give time for community feedback and consensus on this proposal. Finalize and officially document the "Component Expectations".
2. **Develop the Component Checklist:**  Create a practical Component Checklist based on the outlined areas above, ensuring it is both user-friendly and comprehensive.
3. **Document Standards and Checklist:**  Publish the Component Expectations and Checklist in prominent locations (repository README/Guide, website documentation).
4. **Implement Review Process:**  Establish the review process for new component contributions and updates, incorporating the Component Checklist as a key evaluation tool.
5. **Retroactively Apply Standards (Where Feasible):**  Evaluate existing components in `pipelines/components` (if any are retained) and prioritize updates to align them with the new standards.
6. **Community Communication and Education:**  Communicate the new standards to the KFP community and provide guidance on how to utilize them.

## Conclusion

Establishing and implementing official component expectations and standards is a critical step towards maturing the KFP component ecosystem and improving the overall user experience. By providing clear guidelines for component structure, quality, usability, and documentation, implementing this proposal can lead to a more reliable and trustworthy KFP platform, which benefits both users and contributors. This focused effort on standards is important to realizing the broader vision of a more user-friendly Kubeflow Pipelines platform for machine learning workflows.