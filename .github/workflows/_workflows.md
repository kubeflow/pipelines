# GitHub Workflows Summary

| Filename | Description | On Push | On PR |
| --- | --- | :---: | :---: |
| [add-ci-passed-label.yml](.github/workflows/add-ci-passed-label.yml) | Adds `ci-passed` label to a PR once the `CI Check` workflow completes successfully. Resets the `ci-passed` label status when a pull request is synchronized or reopened, indicating that changes have been pushed and CI needs to rerun. |  |  |
| [api-server-tests.yml](.github/workflows/api-server-tests.yml) | Runs e2e tests to verify API Server REST Endpoints. | ✓ | ✓ |
| [backend-visualization.yml](.github/workflows/backend-visualization.yml) | Runs unit tests against the backend visualization code. | ✓ | ✓ |
| [build-and-push.yml](.github/workflows/build-and-push.yml) | Builds and pushes images. |  |  |
| [ci-checks.yml](.github/workflows/ci-checks.yml) | Checks if all CI gates have passed by polling every 5 minutes for a total of 8 attempts. |  |  |
| [codeql.yml](.github/workflows/codeql.yml) | Runs CodeQL static analysis against the codebase. |  |  |
| [docs-freshness.yml](.github/workflows/docs-freshness.yml) | Verifies that the Argo Workflows version compatibility information in README.md stays in sync with the actual versions used in the codebase. | ✓ | ✓ |
| [e2e-test.yml](.github/workflows/e2e-test.yml) | Runs e2e tests. | ✓ | ✓ |
| [first-interaction.yml](.github/workflows/first-interaction.yml) | Welcomes first time contributors to the project. |  |  |
| [frontend.yml](.github/workflows/frontend.yml) | Runs unit tests against the frontend. | ✓ | ✓ |
| [gcpc-modules-tests.yml](.github/workflows/gcpc-modules-tests.yml) | Runs unit tests against the Google Cloud Pipeline Component logic. | ✓ | ✓ |
| [image-builds-master.yml](.github/workflows/image-builds-master.yml) | Builds and pushes images for master. | ✓ |  |
| [image-builds-release.yml](.github/workflows/image-builds-release.yml) | Builds and pushes images for release. |  |  |
| [image-builds-with-cache.yml](.github/workflows/image-builds-with-cache.yml) | Builds and caches images. |  |  |
| [kfp-kubernetes-execution-tests.yml](.github/workflows/kfp-kubernetes-execution-tests.yml) | Runs e2e tests against the kfp-kubernetes SDK. | ✓ | ✓ |
| [kfp-kubernetes-library-test.yml](.github/workflows/kfp-kubernetes-library-test.yml) | Runs unit tests against the kfp-kubernetes SDK. | ✓ | ✓ |
| [kfp-samples.yml](.github/workflows/kfp-samples.yml) | Runs e2e tests against the samples directory. | ✓ | ✓ |
| [kfp-sdk-runtime-tests.yml](.github/workflows/kfp-sdk-runtime-tests.yml) | Runs unit tests against sdk/runtime_tests. | ✓ | ✓ |
| [kfp-sdk-tests.yml](.github/workflows/kfp-sdk-tests.yml) | Runs unit tests against the kfp SDK. | ✓ | ✓ |
| [kfp-webhooks.yml](.github/workflows/kfp-webhooks.yml) | Validates KFP across multiple versions of Kubernetes. | ✓ | ✓ |
| [kubeflow-pipelines-manifests.yml](.github/workflows/kubeflow-pipelines-manifests.yml) | Validates KFP manifests. | ✓ | ✓ |
| [periodic.yml](.github/workflows/periodic.yml) | Runs functional tests every day at midnight. |  |  |
| [pr-commands.yml](.github/workflows/pr-commands.yml) | Processes PR commands. |  |  |
| [pre-commit.yml](.github/workflows/pre-commit.yml) | Runs the pre-commit hooks against the codebase. | ✓ | ✓ |
| [presubmit-backend.yml](.github/workflows/presubmit-backend.yml) | Runs unit tests against the Golang backend. | ✓ | ✓ |
| [readthedocs-builds.yml](.github/workflows/readthedocs-builds.yml) | Validates KFP Readthedocs release readiness. | ✓ | ✓ |
| [sdk-component-yaml.yml](.github/workflows/sdk-component-yaml.yml) | Validates component loading. | ✓ | ✓ |
| [sdk-docformatter.yml](.github/workflows/sdk-docformatter.yml) | Runs doc-formatter tests. | ✓ | ✓ |
| [sdk-execution.yml](.github/workflows/sdk-execution.yml) | Runs e2e tests against kfp SDK. | ✓ | ✓ |
| [sdk-isort.yml](.github/workflows/sdk-isort.yml) | Runs isort tests. | ✓ | ✓ |
| [sdk-upgrade.yml](.github/workflows/sdk-upgrade.yml) | Validates SDK upgrade succeeds. | ✓ | ✓ |
| [sdk-yapf.yml](.github/workflows/sdk-yapf.yml) | Runs YAPF against SDK. | ✓ | ✓ |
| [stale.yml](.github/workflows/stale.yml) | Warns and then closes issues and PRs that have had no activity for a specified amount of time. |  |  |
| [trivy.yml](.github/workflows/trivy.yml) | Runs trivy vulnerability scanner in repo mode and uploads results to GitHub Security tab. | ✓ | ✓ |
| [upgrade-test.yml](.github/workflows/upgrade-test.yml) | Validates KFP upgrade succeeds. | ✓ | ✓ |
| [validate-generated-files.yml](.github/workflows/validate-generated-files.yml) | Validates automatically generated files. | ✓ | ✓ |
