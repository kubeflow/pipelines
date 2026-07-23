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

from kfp import dsl, compiler
from kfp.dsl import Input, Output, Dataset, Model, Metrics


@dsl.component(
    base_image='python:3.9',
    packages_to_install=['numpy', 'pillow']
)
def fetch_image_batch(
    num_images: int,
    images: Output[Dataset]
):
    """Simulates pulling a compressed batch of image arrays from an object store
    and unpacking them into a target staging path.

    Args:
        num_images: Number of synthetic images to generate.
        images: Output dataset directory where images will be unpacked.
    """
    import os
    import numpy as np
    from PIL import Image

    # Ensure the directory exists
    os.makedirs(images.path, exist_ok=True)

    # Generate synthetic images (e.g. random noise or simple shapes)
    for i in range(num_images):
        # Create a 256x256 RGB image of random noise
        img_array = np.random.randint(0, 256, (256, 256, 3), dtype=np.uint8)
        
        # Occasionally inject a "structural anomaly" (e.g., a bright solid square)
        if i % 5 == 0:
            img_array[50:150, 50:150, :] = 255
            
        img = Image.fromarray(img_array)
        img.save(os.path.join(images.path, f"image_{i}.png"))


@dsl.component(
    base_image='python:3.9',
    packages_to_install=['numpy', 'pandas', 'pillow']
)
def extract_visual_features(
    images: Input[Dataset],
    features: Output[Dataset]
):
    """Iterates through the raw image paths, resizes them to standard dimensions (224x224),
    and flattens the arrays into structural feature representations.

    Args:
        images: Input dataset directory containing raw image files.
        features: Output dataset path to save the extracted flattened feature matrix.
    """
    import os
    import time
    import numpy as np
    from PIL import Image

    start_time = time.time()
    
    image_files = [f for f in os.listdir(images.path) if f.endswith('.png')]
    feature_list = []
    
    for filename in sorted(image_files):
        img_path = os.path.join(images.path, filename)
        with Image.open(img_path) as img:
            # Resize image to standard dimensions (224x224)
            img_resized = img.resize((224, 224))
            # Convert to grayscale to keep feature dimension reasonable
            img_gray = img_resized.convert('L')
            # Flatten the array
            flat_arr = np.array(img_gray).flatten()
            feature_list.append(flat_arr)
            
    feature_matrix = np.array(feature_list)
    
    # Save the features as a .npy file
    np.save(features.path, feature_matrix)
    
    execution_time = time.time() - start_time
    features.metadata['execution_time_seconds'] = float(execution_time)
    features.metadata['num_records'] = int(feature_matrix.shape[0])
    features.metadata['feature_dimension'] = int(feature_matrix.shape[1])


@dsl.component(
    base_image='python:3.9',
    packages_to_install=['numpy', 'scikit-learn']
)
def train_anomaly_detector(
    features: Input[Dataset],
    model: Output[Model],
    metrics: Output[Metrics],
    contamination: float = 0.2
):
    """Loads the extracted features, fits an Isolation Forest model to detect visual anomalies,
    and logs model artifacts and training execution metrics.

    Args:
        features: Input dataset containing extracted flattened visual features.
        model: Output model artifact.
        metrics: Output metrics artifact.
        contamination: The proportion of outliers in the data set.
    """
    import time
    import pickle
    import numpy as np
    from sklearn.ensemble import IsolationForest

    start_time = time.time()

    # Load feature matrix
    feature_matrix = np.load(features.path)

    # Initialize and fit Isolation Forest classifier
    clf = IsolationForest(
        contamination=contamination,
        random_state=42
    )
    clf.fit(feature_matrix)

    # Save the trained Isolation Forest model
    with open(model.path, 'wb') as f:
        pickle.dump(clf, f)

    # Calculate anomaly scores (Isolation Forest decision_function outputs negative anomaly scores)
    # The lower the score, the more abnormal the image
    scores = clf.decision_function(feature_matrix)
    avg_anomaly_score = float(np.mean(scores))

    execution_time = time.time() - start_time

    # Log metrics to native KFP Metrics artifact
    metrics.log_metric('training_execution_time_seconds', float(execution_time))
    metrics.log_metric('average_anomaly_score', avg_anomaly_score)

    # Populate metadata on model
    model.metadata['framework'] = 'scikit-learn'
    model.metadata['algorithm'] = 'IsolationForest'
    model.metadata['contamination'] = contamination


@dsl.pipeline(
    name='image-anomaly-detection-pipeline',
    description='A production-grade KFP v2 image processing and anomaly detection pipeline.'
)
def image_anomaly_pipeline(
    num_images: int = 20,
    contamination: float = 0.2
):
    """A pipeline to simulate image batch downloading, resizing/flattening, and anomaly model training."""
    
    # 1. Fetch/simulate batch of images
    fetch_task = fetch_image_batch(num_images=num_images)
    
    # 2. Extract features (resize, grayscale, flatten)
    extract_task = extract_visual_features(images=fetch_task.outputs['images'])
    
    # 3. Train Isolation Forest model
    train_task = train_anomaly_detector(
        features=extract_task.outputs['features'],
        contamination=contamination
    )


if __name__ == '__main__':
    compiler.Compiler().compile(image_anomaly_pipeline, __file__ + '.yaml')
