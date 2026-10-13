Create a complete KFP v2 image processing and embedding pipeline named 'pipeline.py'

The pipeline must model these sequential operations:
1. fetch_image_batch: Simulates pulling a compressed batch of image arrays from an object store and unpacking them into a target staging path.
2. extract_visual_features: Iterates through the raw image paths, resizes them to standard dimensions (e.g., 224x224), flattens the arrays into structural feature matrices, and outputs the directory path.
3. train_anomaly_detector: Loads the extracted features from the previous step to fit an Isolation Forest or simple classifier to detect structural visual anomalies, logging the total execution time and final anomaly score.
