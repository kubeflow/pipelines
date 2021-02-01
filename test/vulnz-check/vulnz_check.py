# %%
from typing import NamedTuple, List
import os

import kfp
from kfp.components import func_to_container_op, InputPath, OutputPath

# %%
# Vulnerability checking


def fetch_image_digest(image: str) -> str:
    """ Fetch digest of an image.

    Args:
        image (str): image url

    Returns:
        str: full image url with sha256 digest
    """
    import subprocess
    import sys
    digest = subprocess.check_output([
        'gcloud', 'container', 'images', 'describe', image,
        "--format=value(image_summary.fully_qualified_digest)"
    ]).decode(sys.stdout.encoding)
    return digest


fetch_image_digest_component = kfp.components.create_component_from_func(
    fetch_image_digest, base_image='google/cloud-sdk:alpine'
)

kritis_check = kfp.components.load_component_from_text(
    '''
name: Kritis Check
inputs:
- {name: Image, type: String}
outputs:
- {name: Vulnerability Report, type: String}
metadata:
  annotations:
    author: Yuan Gong <gongyuan94@gmail.com>
implementation:
  container:
    image: gcr.io/gongyuan-pipeline-test/kritis-signer
    command:
    - /bin/bash
    - -exc
    - |
      set -o pipefail

      export PATH=/bin # this image does not have PATH defined
      mkdir -p "$(dirname "$1")";
      mkdir -p /workspace
      cd /workspace
      cat >policy.yaml <<EOF
      apiVersion: kritis.grafeas.io/v1beta1
      kind: VulnzSigningPolicy
      metadata:
        name: kfp-vsp
      spec:
        imageVulnerabilityRequirements:
          maximumFixableSeverity: MEDIUM
          maximumUnfixableSeverity: HIGH
          allowlistCVEs:
          - projects/goog-vulnz/notes/CVE-2019-19814
      EOF
      cat policy.yaml

      /kritis/signer \
        -v=10 \
        -logtostderr \
        -image="$0" \
        -policy="policy.yaml" \
        -mode=check-only \
        |& tee "$1"

    - inputValue: Image
    - outputPath: Vulnerability Report
'''
)


@func_to_container_op
def kfp_images(version: str, registry_url: str) -> str:
    images = [
        'persistenceagent', 'scheduledworkflow', 'frontend',
        'viewer-crd-controller', 'visualization-server', 'inverse-proxy-agent',
        'metadata-writer', 'cache-server', 'cache-deployer', 'metadata-envoy'
    ]
    import json
    return json.dumps([
        '{}/{}:{}'.format(registry_url, image, version) for image in images
    ])


# Combining all pipelines together in a single pipeline
def vulnz_check_pipeline(
        version: str = '1.3.0',
        registry_url: str = 'gcr.io/gongyuan-pipeline-test/dev'
):
    kfp_images_task = kfp_images(registry_url=registry_url, version=version)
    with kfp.dsl.ParallelFor(kfp_images_task.output) as image:
        fetch_image_digest_task = fetch_image_digest_component(image=image)
        vulnz_check_task = kritis_check(image=fetch_image_digest_task.output)


# %%
if __name__ == '__main__':
    kfp_host = os.getenv('KFP_HOST')
    if kfp_host is None:
        print('KFP_HOST env var is not set')
        exit(1)
    client = kfp.Client(host=kfp_host)
    # Submit the pipeline for execution:
    run = client.create_run_from_pipeline_func(
        vulnz_check_pipeline, arguments={'version': '1.3.0'}
    )
    print('Run details:')
    print('{}/#/runs/details/{}'.format(kfp_host, run.run_id))
    timeout_in_seconds = 5 * 60
    run_result = client.wait_for_run_completion(run.run_id, timeout_in_seconds)
    print(run_result.run.status)  # Failed or ...

    # Compiling the pipeline
    # kfp.compiler.Compiler().compile(vulnz_check_pipeline, __file__ + '.yaml')

# %%
