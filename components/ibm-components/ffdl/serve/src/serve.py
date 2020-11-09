# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import argparse
from pathlib import Path

from app import run_safe

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--model_id', type=str, help='Training model id')
    parser.add_argument('--deployment_name', type=str, help='Deployment name for the seldon service')
    parser.add_argument('--model_class_name', type=str, help='PyTorch model class name', default='ModelClass')
    parser.add_argument('--model_class_file', type=str, help='File that contains the PyTorch model class', default='model_class.py')
    parser.add_argument('--serving_image', type=str, help='Model serving images', default='aipipeline/seldon-pytorch:0.1')
    parser.add_argument('--output_deployment_result_path', type=str, help='Output path for deployment result', default='/tmp/deployment_result.txt')
    args = parser.parse_args()

    with open("/app/secrets/s3_url", 'r') as f:
        s3_url = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/result_bucket", 'r') as f:
        bucket_name = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/s3_access_key_id", 'r') as f:
        s3_access_key_id = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/s3_secret_access_key", 'r') as f:
        s3_secret_access_key = f.readline().strip('\'')
    f.close()
    with open("/app/secrets/k8s_public_nodeport_ip", 'r') as f:
        seldon_ip = f.readline().strip('\'')
    f.close()

    model_id = args.model_id
    deployment_name = args.deployment_name
    model_class_name = args.model_class_name
    model_class_file = args.model_class_file
    serving_image = args.serving_image

    formData = {
        "public_ip": seldon_ip,
        "aws_endpoint_url": s3_url,
        "aws_access_key_id": s3_access_key_id,
        "aws_secret_access_key": s3_secret_access_key,
        "training_results_bucket": bucket_name,
        "model_file_name": "model.pt",
        "deployment_name": deployment_name,
        "training_id": model_id,
        "container_image": serving_image,
        "check_status_only": False,
        "model_class_name": model_class_name,
        "model_class_file": model_class_file
    }

    metrics = run_safe(formData, "POST")
    print(metrics)

    Path(args.output_deployment_result_path).parent.mkdir(parents=True, exist_ok=True)
    Path(args.output_deployment_result_path).write_text(json.dumps(metrics))

    print('\nThe Model is running at ' + metrics['deployment_url'])
