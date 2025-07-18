# Revolutionizing Kubeflow Pipeline Testing: A 2025 Proposal

<!-- TOC -->
* [Revolutionizing Kubeflow Pipeline Testing: A 2025 Proposal](#revolutionizing-kubeflow-pipeline-testing-a-2025-proposal)
  * [Summary](#summary)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [Current Testing Process](#current-testing-process)
  * [Proposed Changes](#proposed-changes)
  * [Testing Strategy](#testing-strategy)
    * [Server API Tests](#server-api-tests)
    * [SDK Tests](#sdk-tests)
      * [Targeted Tests](#targeted-tests)
        * [Compilation Tests](#compilation-tests)
        * [API Tests](#api-tests)
      * [Semi Exploratory Tests](#semi-exploratory-tests)
      * [Full Exploratory Tests](#full-exploratory-tests)
    * [End to End Tests](#end-to-end-tests)
      * [Critical Regression Testing](#critical-regression-testing)
      * [Full Regression Testing](#full-regression-testing)
  * [Test Architecture Changes](#test-architecture-changes)
    * [Test Code Architecture](#test-code-architecture)
  * [Benefits of Proposed Changes](#benefits-of-proposed-changes)
  * [Next Steps](#next-steps)
  * [POC](#poc)
      * [Server API Tests:](#server-api-tests-1)
    * [SDK Compilation Tests:](#sdk-compilation-tests)
<!-- TOC -->

## Summary
To boost the release efficiency of the Kubeflow Pipelines project and increase confidence in our Pull Requests (PRs), we must prioritize tests that verify functional changes at the service/component level. We aim to enhance test coverage transparency, broaden coverage beyond basic positive scenarios, and implement multi-tiered testing throughout the Software Development Life Cycle (SDLC). We also need user-friendly test reports for quick coverage assessment and simplified debugging (eliminating the need to sift through log files). Our tests should be structured and grouped logically for easy understanding.

This proposal outlines changes to our testing approach, emphasizing improved test efficacy, reduced testing time, enhanced confidence, and more targeted functional testing. It also explores leveraging existing data for real-world end-to-end testing scenarios and developing a scalable testing framework for broader use and integration.
## Goals
1. Comprehensive test coverage for Kubeflow Pipelines APIs.
2. Main list of pipeline files - A repository of diverse valid and invalid pipeline files (organized in a single dedicated directory) sourced from existing tests, customer contributions, user scenarios.
3. A standardized test framework for all contributors: Ginkgo + Gomega for all tests except SDK Unit/Component tests, which will use Pytest and front end Javascript tests.
4. Refactoring of existing v2 tests, to be conducted in phases with specific goals for each phase.
5. Cleanup and reorganization of test code to eliminate redundancy.
6. Documentation improvements:
   1. A new Test Process strategy document will be added to the CONTRIBUTING guide.
   2. Test code documentation with examples of creating new test cases.
   3. All Tests should be environment agnostic, i.e. these tests should be able to run in any type of cluster Kind, Minikube, Cloud and on any namespace
   4. Test code guidelines will be added (such as Ginkgo description, logging etc.)
7. Current Linting styles will be kept intact and Python SDK tests will be run through pylint to make sure it passes
8. Test Coverage reporting will be produced as part of either a new or existing workflow
9. Max CI workflow time will be kept to 1 hr, any workflow taking > 1 hr will be split into multiple workflows

## Non-Goals
1. We will not initially cover components that users can not directly interact with . For e.g. “Launcher/Executor”, "Driver", we will still add indirect coverage for this pipeline service/component but until we have a direct way to interact with this, direct functional coverage will be out of scope.
2. Stable third-party dependencies, such as Argo Workflows, will not be included in this initiative.
3. Front End Tests will not be refactored
4. [Semi Exploratory E2E tests](#semi-exploratory-tests) will not be considered in this effort
5. AI-generated pipeline specs (using Gemini, ChatGPT, or Cursor) will not be included as part of this

# Proposal
This section describes the proposed changes to our current testing processes and supporting test architecture. These changes aim to improve the quality, reliability, and speed of our testing, leading to better product outcomes.

## Current Testing Process
Our current testing process includes:
* **Unit Testing**: Good coverage, as reflected in test coverage reports.
* **Integration Testing**: API tests - Focuses on basic positive scenarios with minimal verification steps. Test code lacks readability and standardization.
* **End-to-End Testing**: Sample Tests and SDK Execution Tests - Covers a few basic pipelines but lacks component state verification, complicating debugging.

## Proposed Changes
The following table outlines the proposed changes to the testing process.

| **Area**      | **Current Process**                                                                              | **Proposed Process**                                                                                                                | **Justification**                                                                                                   |
|-----------|----------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------|
| Automation | Limited functional coverage, indirect testing, inconsistent code, unreadable/unmaintainable code | Increase functional coverage, standardize framework, readable/maintainable code, negative test coverage, organize documentation | Improved testing, higher confidence, easier coverage expansion with a standardized framework, maintainable code |
| Testing Process | Lack of transparency, disorganized tests, no test plan reviews  | Test plans/requirements included in design reviews, merge strategy with sign-off  | No post-merge surprises, testable code, quality-first approach  |                                                                                              |                                                                                                                                 |                                                                                                                 |
| Feedback Loop | Hard to debug test failures, no test reports and failures do not correspond to any severity | Better logging & code doc, improved reporting for easier debugging  | Easier to debug failures, easily digestible reporting by devs and non-developers as well  |                                                                                                |                                                                                                                                 |                                                                                                                 |

## Testing Strategy
Before we get into the test strategy, let's revisit the project architecture as described [here](https://github.com/kubeflow/pipelines/blob/master/docs/sdk/Architecture.md). If we have to describe the workflow in short, it would that, user interact with KFP  via APIs, data is persisted in DB, and based on the endpoint, we invoke different components, like Argo Workflows which schedules pods, that runs a specific action, persists data in DB

### Server API Tests
Validation output of the service/components that a specific endpoint interacts with provides direct comprehensive functional coverage. To visualize this, please see below:
![Overall Test Strategy.png](Overall%20Test%20Strategy.png)

And to further explain it, lets take an example of PipelineUploadAPI, the workflow for this API is as follows:
![KFP PipelineUpload API Workflow.png](KFP%20PipelineUpload%20API%20Workflow.png)

And the proposed testing strategy for this would be:
![PipelineUpload API Test Strategy.png](PipelineUpload%20API%20Test%20Strategy.png)

### SDK Tests
Lightweight SDK tests focused on compiled output validation, with opportunities to leverage Pytest parameterized tests and other techniques for increased coverage. In order to achieve this, we will have define DSL components as independent objects that can be referenced in the tests for compilation. And with this approach, we can achieve multiple different type of SDK test coverage:
#### Targeted Tests
##### Compilation Tests
![SDK testing - Targeted.png](SDK%20testing%20-%20Targeted.png)


Using Pytest’s parameterized feature, we can define components as parameters to a compile function, whose output is then validated against a known yaml file specified by an expected parameter.

**Pros:**
* Code reusage - for components
* Targeted testing
* Easy expansion of tests
* No need to spin up a KFP server

**Cons:**
* As a developer, we need to come up with combinations of components making up a pipeline
* Expected valid YAML files needs to be generated
* However we already have a lot of DSL Components defined that are used in different tests. And we can leverage existing passing tests to generate valid YAML for us as well.

##### API Tests

~~**Option 1:**

![SDK API Option 1.png](SDK%20API%20Option%201.png)

Add a proxy to capture all Http calls to the API server, and validate if the request is valid or not. This way, we won’t need a KFP service running in a cluster, and we can verify SDK tests in a standalone environment~~


**Option 2:**

![SDK API Option 2.png](SDK%20API%20Option%202.png)

Do not add a proxy but instead let the API server validate the request and we just validate the response from the API service to confirm if SDK made the right request or not.

**NOTE:** The suggestion here is choose only 1 option, and I personally align towards **Option 2** as it has less overhead from implementation and maintenance pov and this also means that we won’t have to recreate the validation logic in the test code.


#### Semi Exploratory Tests
Leveraging existing independent components, and using graphical representation of a make up of a pipeline, we can use tools like Graphwalker to perform semi exploratory tests as shown below:

![SDK testing - Semi Exploratory.png](SDK%20testing%20-%20Semi%20Exploratory.png)

**Pros:**
* Provides some auto exploratory testing
* Semi Self Managed - Since there is no expected outcome, and the validation is against KFP API, the tests can be as random as possible
* Increases our test coverage beyond known use cases

**Cons:**
* Developers will have to create connection points between different components
* Time and cluster resource utilization may be high here, so we will need to limit the number of paths to traverse and time these tests will take

**NOTE**: Sometime components make assumptions about environments. A common example would be expecting environment variables to be present, and those are (today) defined at @pipeline decorator scope. Nested pipelines may be able to help here.

#### Full Exploratory Tests
Using an AI tool, we can compile a pipeline and validate it against the Pipeline Upload API (so basically, leverage Semi Exploratory tests but use an AI tool to generate a YAML):
![SDK testing - Full Exploratory.png](SDK%20testing%20-%20Full%20Exploratory.png)

**Pros**:
* Fully self managed tests
* Provides much broader coverage of edge/non edge cases that we may not have covered in targeted/semi targeted tests

**Cons**
* We may have to build a custom SDG - out of the box models may not work for us
* Potentially duplicate coverage
* Fail on edge cases that are of trivial priority

### End to End Tests
#### Critical Regression Testing
Using a subset of the main list of pipeline files (already used for API tests and SDK tests), we can run full end to end tests that will confirm the integration of different components that make up KFP.
![End2End-DuringPR.png](End2End-DuringPR.png)

#### Full Regression Testing
Develop a simple application to run pipeline files from a directory (local or remote), run them in parallel, and verify successful execution.
![End2End-Full Regression.png](End2End-Full%20Regression.png)

**Pros**:
* An ad hoc way for us to load the system
* Full end to end testing
* Lot of indirect testing of components without really worrying about the internals
* Easy to expand - all it will take is to generate a new pipeline file
* Flexible testing app for other products that integrates with KFP
* Easy integration with AI tools to perform some exploratory testing

**Cons**:
* Failed pipelines may get hard to debug
* Resource utilization will be high
* Some pipelines may not be possible due to resource constraints
* Long running tests (even after parallelization)
* Parallel tests can cause storage to fill up on the GitHub workers, since we clean up only after the pipeline run finishes. So we may need to explore running tests in parallel on separate GitHub workers/workflows.

## Test Architecture Changes
* Implement all tests in Go using Ginkgo + Gomega, providing BDD-style, readable, and organized test scripts with multi-format output and parallel execution.
* Use Pytest for SDK tests
* Sample tests will be renamed to End to End Tests and labeled with “Critical” label. These tests will be a subset of full regression tests covering critical use cases, and will run with every PR to confirm regression in integration. These tests will also be converted to Ginkgo tests.
* Full Regression (Lightweight End-to-end) Tests will run on a schedule May be start with every merge to master and then change frequency as we gain confidence and as the list of tests grow, to daily, weekly or per release. Go.mod file will be added to these tests to allow reuse of this code at other places.
* Front End Tests will stay the same

### Test Code Architecture

Utilize Ginkgo for API tests, aligning with the primarily Go codebase. And a test should follow the following pattern:
![Test Code Architecture.png](Test%20Code%20Architecture.png)

## Benefits of Proposed Changes

The proposed changes will result in the following benefits:

* Increased testing efficiency and speed.
* Improved test coverage and quality.
* Reduced risk of release failures.
* Enhanced team collaboration and communication.
* Tests become self-documenting if we use Ginkgo

## Next Steps
We recommend the following next steps:

* Conduct a detailed analysis of the proposed changes.
* Develop an implementation plan with timelines and resources.
* Begin pilot testing the new processes and architecture.

## POC
#### Server API Tests:
A POC of new test architecture using Ginkgo + Gomega and testing PipelineUpload API tests is available [here](https://github.com/kubeflow/pipelines/pull/11956)

### SDK Compilation Tests:
[WIP] A simple POC for SDK compilation tests that uses dsl components as pytest parameters, [here](https://github.com/kubeflow/pipelines/pull/11983)

## Miscellaneous
Going forward, we can start to measure the success of this effort by measuring following metrics:
1. Measuring number of NEW issues reported after every release is a great measure of success/confidence
2. Git workflow history - but we can measure trends of post merge failures
3. Number of new automated tests per release
