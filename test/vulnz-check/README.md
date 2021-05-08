# Vulnerability Check Pipeline

Background context that explored the process we want and the options we have: [#3857](https://github.com/kubeflow/pipelines/issues/3857).

The pipeline uses Kritis Signer to check vulnerabilities using Google Cloud vulnerability scanning.

For reference, [we can use Kritis Signer to check vulnerabilities using a policy](https://cloud.google.com/binary-authorization/docs/creating-attestations-kritis#check-only).

There are two pipelines in this folder:

* `mirror_images.py` pipeline is a helper to mirror images to someone's own gcr registry, because Google Cloud vulnerability scanning can only be used in an owned registry.
* `vulnz_check.py` pipeline checks vulnerability against a predefined policy and allowlist.
