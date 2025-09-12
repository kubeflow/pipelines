# GitHub Workflows Summary

| Filename | Description | Triggers |
| --- | --- | --- |
| [add-ci-passed-label.yml](add-ci-passed-label.yml) | Adds `ci-passed` label to a PR once the `CI Check` workflow completes successfully. Resets the `ci-passed` label when a pull request is synchronized or reopened, indicating that changes have been pushed and CI needs to rerun. | workflow_run |
| [api-server-tests.yml](api-server-tests.yml) | Runs e2e tests to verify API Server REST Endpoints. | pull_request, push, workflow_dispatch |
| [backend-visualization.yml](backend-visualization.yml) | Runs unit tests against the backend visualization code. | pull_request, push |
| [build-and-push.yml](build-and-push.yml) | Builds and pushes images. | workflow_call, workflow_dispatch |
| [ci-checks.yml](ci-checks.yml) | Checks if all CI gates have passed by polling every 5 minutes for a total of 8 attempts. | pull_request_target |
| [codeql.yml](codeql.yml) | Runs CodeQL static analysis against the codebase. | schedule |
| [docs-freshness.yml](docs-freshness.yml) | Verifies that the Argo Workflows version compatibility information in README.md stays in sync with the actual versions used in the codebase. | pull_request, push |
| [e2e-test.yml](e2e-test.yml) | Runs e2e tests. | pull_request, push |
| [first-interaction.yml](first-interaction.yml) | Welcomes first time contributors to the project. | issues, pull_request_target |
| [frontend.yml](frontend.yml) | Runs unit tests against the frontend. | pull_request, push |
| [gcpc-modules-tests.yml](gcpc-modules-tests.yml) | Runs unit tests against the Google Cloud Pipeline Component logic. | pull_request, push |
| [image-builds-master.yml](image-builds-master.yml) | Builds and pushes images for master. | push |
| [image-builds-release.yml](image-builds-release.yml) | Builds and pushes images for release. | workflow_dispatch |
| [image-builds-with-cache.yml](image-builds-with-cache.yml) | Builds and caches images. | workflow_call |
| [kfp-kubernetes-execution-tests.yml](kfp-kubernetes-execution-tests.yml) | Runs e2e tests against the kfp-kubernetes SDK. | pull_request, push |
| [kfp-kubernetes-library-test.yml](kfp-kubernetes-library-test.yml) | Runs unit tests against the kfp-kubernetes SDK. | pull_request, push |
| [kfp-samples.yml](kfp-samples.yml) | Runs e2e tests against the samples directory. | pull_request, push |
| [kfp-sdk-runtime-tests.yml](kfp-sdk-runtime-tests.yml) | Runs unit tests against sdk/runtime_tests. | pull_request, push |
| [kfp-sdk-tests.yml](kfp-sdk-tests.yml) | Runs unit tests against the kfp SDK. | pull_request, push |
| [kfp-webhooks.yml](kfp-webhooks.yml) | Validates KFP across multiple versions of Kubernetes. | pull_request, push |
| [kubeflow-pipelines-manifests.yml](kubeflow-pipelines-manifests.yml) | Validates KFP manifests. | pull_request, push |
| [periodic.yml](periodic.yml) | Runs functional tests every day at midnight. | schedule |
| [pr-commands.yml](pr-commands.yml) | Processes PR commands. | issue_comment |
| [pre-commit.yml](pre-commit.yml) | Runs the pre-commit hooks against the codebase. | pull_request, push |
| [presubmit-backend.yml](presubmit-backend.yml) | Runs unit tests against the Golang backend. | pull_request, push |
| [readthedocs-builds.yml](readthedocs-builds.yml) | Validates KFP Readthedocs release readiness. | pull_request, push |
| [sdk-component-yaml.yml](sdk-component-yaml.yml) | Validates component loading. | pull_request, push |
| [sdk-docformatter.yml](sdk-docformatter.yml) | Runs doc-formatter tests. | pull_request, push |
| [sdk-execution.yml](sdk-execution.yml) | Runs e2e tests against kfp SDK. | pull_request, push |
| [sdk-isort.yml](sdk-isort.yml) | Runs isort tests. | pull_request, push |
| [sdk-upgrade.yml](sdk-upgrade.yml) | Validates SDK upgrade succeeds. | pull_request, push |
| [sdk-yapf.yml](sdk-yapf.yml) | Runs YAPF against SDK. | pull_request, push |
| [stale.yml](stale.yml) | Warns and then closes issues and PRs that have had no activity for a specified amount of time. | schedule |
| [trivy.yml](trivy.yml) | Runs trivy vulnerability scanner in repo mode and uploads results to GitHub Security tab. | pull_request, push |
| [upgrade-test.yml](upgrade-test.yml) | Validates KFP upgrade succeeds. | pull_request, push |
| [validate-generated-files.yml](validate-generated-files.yml) | Validates automatically generated files. | pull_request, push |
