#!/bin/sh

# Run conformance test and generate test report.
./api-test \
    -namespace kubeflow \
    -runIntegrationTests=true \
    -isDevMode=false \
    -isKubeflowMode=true \
    -resourceNamespace=kf-conformance \
    -test.v \
    ${ADDITIONAL_FLAGS} \
    2>&1 | tee /tmp/kfp-conformance.log

# Create the done file.
touch /tmp/kfp-conformance.done

echo "Done..."
# Keep the container running so the test logs can be downloaded.
while true; do sleep 10000; done