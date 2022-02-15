```
SESSION="MTY0NDkzNDc0N3xOd3dBTkVOVFRqSlhWazVEU2s1VVJGWlhTRVZJUVVwWVZsaElVMFpQUWxSTU1rRklOVWxhVDFkTFJFNUhVME0zVWxCRFJVNUVWbEU9fPL4-4KasyG2hFk0qAsl0--5DcwDVAuL-nCfFvajL7W1"
RUN_ID="c9af6a77-14a3-4500-8a58-933294994f3e"
NODE_ID="bash-pipeline-nxgtk-3048189797"
# https://istio-ingressgateway-istio-system.apps.p027.otc.mcs-paas.io/pipeline/artifacts/minio/admin/artifacts/bash-pipeline-v8thz/2022/02/15/bash-pipeline-v8thz-1355294551/mlpipeline-metrics.tgz\?namespace\=admin
# artifacts/bash-pipeline-v8t/2022/02/15/bash-pipeline-v8thz-1355294551/mlpipeline-metrics.tgz
ARTIFACT_NAME="mlpipeline-metrics"
curl -s -H "Cookie: authservice_session=$SESSION" \
"https://istio-ingressgateway-istio-system.apps.p027.otc.mcs-paas.io/pipeline/apis/v1beta1/runs/${RUN_ID}/nodes/${NODE_ID}/artifacts/${ARTIFACT_NAME}:read?resource_reference_key.type=NAMESPACE&resource_reference_key.id=admin"
```
