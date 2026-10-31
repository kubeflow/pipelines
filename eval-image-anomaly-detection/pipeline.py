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
"""Production-grade KFP v2 Image Processing, Feature Extraction, and Anomaly
Detection Pipeline."""

from kfp import compiler
from kfp import dsl


@dsl.component(
    base_image='python:3.11',
    packages_to_install=['numpy', 'pillow'],
)
def fetch_image_batch(
    raw_images_dir: dsl.Output[dsl.Artifact],
    num_images: int = 20,
    random_state: int = 42,
):
    """Simulates pulling a compressed batch of image arrays from an object
    store and unpacking them into a target staging path.

    Args:
        raw_images_dir: Output artifact directory representing the target
          staging path.
        num_images: Total number of synthetic images to simulate in the batch.
        random_state: Random state seed for reproducible image generation.
    """
    import io
    import os
    import zipfile

    import numpy as np
    from PIL import Image

    np.random.seed(random_state)
    os.makedirs(raw_images_dir.path, exist_ok=True)

    # Simulate pulling a compressed batch archive from an object store
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, 'w', zipfile.ZIP_DEFLATED) as zip_file:
        for i in range(num_images):
            # Synthesize standard background vs structural anomaly distributions
            if i % 5 == 0:
                # Structural anomaly: high-contrast distinct intensity patch
                arr = np.random.randint(
                    200, 256, size=(100, 100, 3), dtype=np.uint8)
            else:
                # Regular background noise image array
                arr = np.random.randint(
                    0, 120, size=(100, 100, 3), dtype=np.uint8)
            img = Image.fromarray(arr)
            img_byte_arr = io.BytesIO()
            img.save(img_byte_arr, format='PNG')
            zip_file.writestr(f'image_{i:03d}.png', img_byte_arr.getvalue())

    # Unpack the compressed batch into the target staging directory
    buffer.seek(0)
    with zipfile.ZipFile(buffer, 'r') as zip_file:
        zip_file.extractall(raw_images_dir.path)

    print(
        f'Successfully pulled and unpacked {num_images} images to {raw_images_dir.path}'
    )


@dsl.component(
    base_image='python:3.11',
    packages_to_install=['numpy', 'pillow', 'pandas'],
)
def extract_visual_features(
    raw_images_dir: dsl.Input[dsl.Artifact],
    features_dir: dsl.Output[dsl.Artifact],
    target_width: int = 224,
    target_height: int = 224,
):
    """Iterates through the raw image paths, resizes them to standard
    dimensions, flattens the arrays into structural feature matrices, and
    outputs the directory path.

    Args:
        raw_images_dir: Input artifact directory holding raw image files.
        features_dir: Output artifact directory for the extracted feature
          matrices.
        target_width: Standard target width for resizing.
        target_height: Standard target height for resizing.
    """
    import os

    import numpy as np
    import pandas as pd
    from PIL import Image

    os.makedirs(features_dir.path, exist_ok=True)

    features_list = []
    filenames = sorted([
        f for f in os.listdir(raw_images_dir.path)
        if f.lower().endswith(('.png', '.jpg', '.jpeg'))
    ])

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
            # Structural feature row: summary distribution statistics plus feature slice
            row = [mean_val, std_val, max_val, min_val
                  ] + flattened[:100].tolist()
            features_list.append(row)

    feature_matrix = pd.DataFrame(features_list)
    features_csv = os.path.join(features_dir.path, 'features.csv')
    feature_matrix.to_csv(features_csv, index=False)
    print(
        f'Extracted feature matrix of shape {feature_matrix.shape} saved to {features_csv}'
    )


@dsl.component(
    base_image='python:3.11',
    packages_to_install=['numpy', 'pandas', 'scikit-learn', 'joblib'],
)
def train_anomaly_detector(
    features_dir: dsl.Input[dsl.Artifact],
    model_artifact: dsl.Output[dsl.Model],
    metrics: dsl.Output[dsl.Metrics],
    contamination: float = 0.2,
    random_state: int = 42,
):
    """Loads the extracted features from the previous step to fit an Isolation
    Forest or simple classifier to detect structural visual anomalies, logging
    the total execution time and final anomaly score.

    Args:
        features_dir: Input artifact directory holding the extracted feature
          CSV.
        model_artifact: Serialized model artifact output.
        metrics: Native KFP Metrics artifact output.
        contamination: Proportion of expected outliers/anomalies in the dataset.
        random_state: Random state seed for reproducibility.
    """
    import os
    import time

    import joblib
    import numpy as np
    import pandas as pd
    from sklearn.ensemble import IsolationForest

    start_time = time.time()

    features_csv = os.path.join(features_dir.path, 'features.csv')
    df = pd.read_csv(features_csv)

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

    # Log metrics to native KFP Metrics artifact
    metrics.log_metric('execution_time_seconds', execution_time)
    metrics.log_metric('final_anomaly_score', mean_anomaly_score)
    metrics.log_metric('anomaly_score', mean_anomaly_score)
    metrics.log_metric('num_anomalies_detected', num_anomalies)

    os.makedirs(os.path.dirname(model_artifact.path), exist_ok=True)
    joblib.dump(clf, model_artifact.path)

    print(f'Execution Time: {execution_time:.4f}s')
    print(f'Anomalies Detected: {num_anomalies}')
    print(f'Final Anomaly Score: {mean_anomaly_score:.4f}')


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
    """Image processing and embedding pipeline orchestrating batch fetching,
    feature extraction, and anomaly detection.

    Args:
        num_images: Number of raw image arrays to simulate in the batch.
        target_width: Width to resize raw images to.
        target_height: Height to resize raw images to.
        contamination: Expected outlier contamination rate for Isolation Forest.
        random_state: Random state seed for reproducibility.
    """
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
