# Argo Workflows runtime pin

`VERSION` selects the upstream Argo installation used by the native deployment.
There is no vendored or patched workflow controller in this repository.

When updating the version, run the native schema/resource tests, the complete
compatibility corpus and the deployed browser acceptance test. The deployment
uses upstream PostgreSQL archive/offloaded status and ordinary Argo memoization.
