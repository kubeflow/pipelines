# Security Policy

## Supported Versions

Kubeflow Pipelines versions are expressed as `X.Y.Z`, where X is the major version,
Y is the minor version, and Z is the patch version, following the
[Semantic Versioning](https://semver.org/) terminology.

The Kubeflow Pipelines project maintains release branches for the most recent two minor releases.
Applicable fixes, including security fixes, may be backported to those two release branches,
depending on severity and feasibility.

Users are encouraged to stay updated with the latest releases to benefit from security patches and
improvements.

## Reporting a Vulnerability

We're extremely grateful for security researchers and users that report vulnerabilities to the
Kubeflow Open Source Community. All reports are thoroughly investigated by Kubeflow projects owners.

You can use the following ways to report security vulnerabilities privately:

- Using the Kubeflow Pipelines repository [GitHub Security Advisory](https://github.com/kubeflow/pipelines/security/advisories/new).
- Using our private Kubeflow Steering Committee mailing list: ksc@kubeflow.org.

Please provide detailed information to help us understand and address the issue promptly.

## Disclosure Process

**Acknowledgment**: We will acknowledge receipt of your report within 10 business days.

**Assessment**: The Kubeflow projects owners will investigate the reported issue to determine its
validity and severity.

**Resolution**: If the issue is confirmed, we will work on a fix and prepare a release.

**Notification**: Once a fix is available, we will notify the reporter and coordinate a public
disclosure.

**Public Disclosure**: Details of the vulnerability and the fix will be published in the project's
release notes and communicated through appropriate channels.

## Prevention Mechanisms

Kubeflow Pipelines employs several measures to prevent security issues:

**Code Reviews**: All code changes are reviewed by maintainers to ensure code quality and security.

**Dependency Management**: Regular updates and monitoring of dependencies (e.g. Dependabot) to
address known vulnerabilities.

**Continuous Integration**: Automated testing and security checks are integrated into the CI/CD pipeline.

**Image Scanning**: Container images are scanned for vulnerabilities.

## Communication Channels

For the general questions please join the following resources:

- Kubeflow [Slack channels](https://www.kubeflow.org/docs/about/community/#kubeflow-slack-channels).

- Kubeflow discuss [mailing list](https://www.kubeflow.org/docs/about/community/#kubeflow-mailing-list).

Please **do not report** security vulnerabilities through public channels.
