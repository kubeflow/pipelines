# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""KFP v2 Image Processing, Feature Extraction, and Anomaly Detection Pipeline."""

import os

from kfp import compiler, dsl

_KFP_PACKAGE_PATH = os.getenv('KFP_PACKAGE_PATH')


@dsl.component(
    kfp_package_path=_KFP_PACKAGE_PATH,
    packages_to_install=['numpy', 'pillow'],
)
def fetch_image_batch(
    raw_images_dir: dsl.Output[dsl.Artifact],
    num_images: int = 20,
    random_state: int = 42,
):
    """Simulates pulling a compressed batch of image arrays from an object store and unpacking them into a target staging path."""
    import os
    import numpy as np
    from PIL import Image

    np.random.seed(random_state)
    os.makedirs(raw_images_dir.path, exist_ok=True)

    for i in range(num_images):
        if i % 5 == 0:
            arr = np.random.randint(200, 256, size=(100, 100, 3), dtype=np.uint8)
        else:
            arr = np.random.randint(0, 120, size=(100, 100, 3), dtype=np.uint8)

        img = Image.fromarray(arr)
        img.save(os.path.join(raw_images_dir.path, f'image_{i:03d}.png'))


@dsl.component(
    kfp_package_path=_KFP_PACKAGE_PATH,
    packages_to_install=['numpy', 'pillow', 'pandas'],
)
def extract_visual_features(
    raw_images_dir: dsl.Input[dsl.Artifact],
    features_dir: dsl.Output[dsl.Artifact],
    target_width: int = 224,
    target_height: int = 224,
):
    """Iterates through raw image paths, resizes them to standard dimensions, flattens into structural feature matrices, and outputs directory path."""
    import os
    import numpy as np
    import pandas as pd
    from PIL import Image

    os.makedirs(features_dir.path, exist_ok=True)

    features_list = []
    filenames = sorted([f for f in os.listdir(raw_images_dir.path) if f.endswith('.png')])

    for fname in filenames:
        img_path = os.path.join(raw_images_dir.path, fname)
        with Image.open(img_path) as img:
            resized_img = img.resize((target_width, target_height))
            arr = np.array(resized_img, dtype=np.float32) / 255.0
            flattened = arr.flatten()
            mean_val = float(np.mean(flattened))
            std_val = float(np.std(flattened))
            max_val = float(np.max(flattened))
            min_val = float(np.min(flattened))
            row = [mean_val, std_val, max_val, min_val] + flattened[:100].tolist()
            features_list.append(row)

    df = pd.DataFrame(features_list)
    df.to_csv(os.path.join(features_dir.path, 'features.csv'), index=False)


@dsl.component(
    kfp_package_path=_KFP_PACKAGE_PATH,
    packages_to_install=['numpy', 'pandas', 'scikit-learn', 'joblib'],
)
def train_anomaly_detector(
    features_dir: dsl.Input[dsl.Artifact],
    model_artifact: dsl.Output[dsl.Model],
    metrics: dsl.Output[dsl.Metrics],
    contamination: float = 0.2,
    random_state: int = 42,
):
    """Loads extracted features to fit an Isolation Forest to detect visual anomalies, logging execution time and anomaly score."""
    import os
    import time
    import joblib
    import numpy as np
    import pandas as pd
    from sklearn.ensemble import IsolationForest

    start_time = time.time()

    features_path = os.path.join(features_dir.path, 'features.csv')
    df = pd.read_csv(features_path)

    clf = IsolationForest(
        contamination=contamination,
        random_state=random_state,
    )
    clf.fit(df)

    predictions = clf.predict(df)
    scores = clf.decision_function(df)

    execution_time = float(time.time() - start_time)
    num_anomalies = int(np.sum(predictions == -1))
    mean_anomaly_score = float(np.mean(scores))

    metrics.log_metric('execution_time_seconds', execution_time)
    metrics.log_metric('num_anomalies_detected', num_anomalies)
    metrics.log_metric('mean_anomaly_score', mean_anomaly_score)

    os.makedirs(os.path.dirname(model_artifact.path), exist_ok=True)
    joblib.dump(clf, model_artifact.path)

    print(f'Execution Time: {execution_time:.4f}s')
    print(f'Anomalies Detected: {num_anomalies}')
    print(f'Mean Anomaly Score: {mean_anomaly_score:.4f}')


@dsl.pipeline(
    name='image-anomaly-detection-pipeline',
    description='KFP v2 image processing, feature extraction, and anomaly detection pipeline.',
)
def image_anomaly_detection_pipeline(
    num_images: int = 20,
    target_width: int = 224,
    target_height: int = 224,
    contamination: float = 0.2,
    random_state: int = 42,
):
    """Orchestrates image batch fetching, feature extraction, and anomaly detector training."""
    fetch_task = fetch_image_batch(
        num_images=num_images,
        random_state=random_state,
    )

    extract_task = extract_visual_features(
        raw_images_dir=fetch_task.outputs['raw_images_dir'],
        target_width=target_width,
        target_height=target_height,
    )

    train_anomaly_detector(
        features_dir=extract_task.outputs['features_dir'],
        contamination=contamination,
        random_state=random_state,
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=image_anomaly_detection_pipeline,
        package_path=__file__ + '.yaml',
    )