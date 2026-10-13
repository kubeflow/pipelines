Create a complete KFP v2 training pipeline for mnist_classification

The pipeline must include the following steps:
1. Download the MNIST dataset from openml using sklearn or torchvision.
2. Normalize the images and flatten them for a classic machine learning model.
3. Train a Random Forest or Support Vector Machine classifier.
4. Evaluate the model, logging distinct metrics for accuracy, precision, and recall into a KFP Metrics artifact, and save a localized evaluation report string.