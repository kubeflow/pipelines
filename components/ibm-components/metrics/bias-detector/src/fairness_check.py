# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import json
import argparse
import os

from fairness import fairness_check


def check_fairness(metrics):
    if abs(metrics['Statistical parity difference']) > 0.1:
        return False
    if abs(metrics['Disparate impact']) < 0.8:
        return False
    if abs(metrics['Equal opportunity difference']) > 0.1:
        return False
    if abs(metrics['Average odds difference']) > 0.1:
        return False
    if abs(metrics['False negative rate difference']) > 0.1:
        return False
    return True


def get_secret(path, default=''):
    try:
        with open(path, 'r') as f:
            cred = f.readline().strip('\'')
        f.close()
        return cred
    except:
        return default


if __name__ == "__main__":

    parser = argparse.ArgumentParser()
    parser.add_argument('--model_id', type=str, help='Training model ID', default="training-dummy")
    parser.add_argument('--metric_path', type=str, help='Path for fairness check output', default="/tmp/fairness.txt")
    parser.add_argument('--fairness_status', type=str, help='Path for fairness status output', default="/tmp/status.txt")
    parser.add_argument('--model_class_file', type=str, help='pytorch model class file', default="PyTorchModel.py")
    parser.add_argument('--model_class_name', type=str, help='pytorch model class name', default="PyTorchModel")
    parser.add_argument('--feature_testset_path', type=str, help='Feature test dataset path in the data bucket', default="processed_data/X_test.npy")
    parser.add_argument('--label_testset_path', type=str, help='Label test dataset path in the data bucket', default="processed_data/y_test.npy")
    parser.add_argument('--protected_label_testset_path', type=str, help='Protected label test dataset path in the data bucket', default="processed_data/p_test.npy")
    parser.add_argument('--favorable_label', type=float, help='Favorable label for this model predictions', default=0.0)
    parser.add_argument('--unfavorable_label', type=float, help='Unfavorable label for this model predictions', default=1.0)
    parser.add_argument('--privileged_groups', type=str, help='Privileged feature groups within this model', default="[{'race': 0.0}]")
    parser.add_argument('--unprivileged_groups', type=str, help='Unprivileged feature groups within this model', default="[{'race': 4.0}]")
    parser.add_argument('--data_bucket_name', type=str, help='Bucket that has the processed data', default="training-data")
    parser.add_argument('--result_bucket_name', type=str, help='Bucket that has the training results', default="training-result")

    object_storage_url = get_secret('/app/secrets/s3_url', 'minio-service:9000')
    object_storage_username = get_secret('/app/secrets/s3_access_key_id', 'minio')
    object_storage_password = get_secret('/app/secrets/s3_secret_access_key', 'minio123')

    args = parser.parse_args()
    metric_path = args.metric_path
    model_id = args.model_id
    fairness_status = args.fairness_status
    model_class_file = args.model_class_file
    model_class_name = args.model_class_name
    feature_testset_path = args.feature_testset_path
    label_testset_path = args.label_testset_path
    protected_label_testset_path = args.protected_label_testset_path
    favorable_label = args.favorable_label
    unfavorable_label = args.unfavorable_label
    data_bucket_name = args.data_bucket_name
    result_bucket_name = args.result_bucket_name
    privileged_groups = eval(args.privileged_groups)
    unprivileged_groups = eval(args.unprivileged_groups)

    metrics = fairness_check(object_storage_url, object_storage_username, object_storage_password,
                             data_bucket_name, result_bucket_name, model_id,
                             feature_testset_path=feature_testset_path,
                             label_testset_path=label_testset_path,
                             protected_label_testset_path=protected_label_testset_path,
                             model_class_file=model_class_file,
                             model_class_name=model_class_name,
                             favorable_label=favorable_label,
                             unfavorable_label=unfavorable_label,
                             privileged_groups=privileged_groups,
                             unprivileged_groups=unprivileged_groups)

    if not os.path.exists(os.path.dirname(metric_path)):
        os.makedirs(os.path.dirname(metric_path))
    with open(metric_path, "w") as report:
        report.write(json.dumps(metrics))
