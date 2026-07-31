@kfp-generator Create a complete KFP v2 image processing and embedding pipeline named 'pipeline.py' under a self-contained evaluation directory.

The pipeline must model these sequential operations:
1. fetch_image_batch: Simulates pulling a compressed batch of image arrays from an object store and unpacking them into a target staging path.
2. extract_visual_features: Iterates through the raw image paths, resizes them to standard dimensions (e.g., 224x224), flattens the arrays into structural feature matrices, and outputs the directory path.
3. train_anomaly_detector: Loads the extracted features from the previous step to fit an Isolation Forest or simple classifier to detect structural visual anomalies, logging the total execution time and final anomaly score.

Ensure the file contains clean docstrings, clear input/output directory variable definitions using the modern KFP v2 DSL decorators, and the standard compilation execution block. Run our local evaluation script with --run --host http://localhost:8080 on it to guarantee a valid KFP v2 YAML specification and verify live backend execution.