
# QSVM Pipeline

This repository contains a Quantum Support Vector Machine (QSVM) pipeline built using Kubeflow Pipelines and Qiskit. The pipeline leverages quantum computing for machine learning tasks by integrating classical machine learning techniques with quantum simulators.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Usage](#usage)
- [Components](#components)
- [License](#license)

## Overview

The **QSVM Pipeline** is designed to run on Kubeflow Pipelines. It uses quantum computing principles through Qiskit to implement a quantum version of the Support Vector Machine (SVM). This allows for experimenting with quantum machine learning models on both classical and quantum simulators.

### Key Features

- **Quantum Support Vector Machine (QSVM)** using Qiskit.
- Classical machine learning operations with **scikit-learn**.
- End-to-end integration with **Kubeflow Pipelines** for scalable execution.
- Simulated quantum computing environment using **Qiskit AerSimulator**.

## Installation

To use this pipeline, make sure you have **Kubeflow Pipelines** set up. The pipeline uses a custom Docker image based on `python:3.10.12` with several dependencies installed, such as Qiskit and scikit-learn.

### Prerequisites

- **Kubeflow Pipelines** installed on your Kubernetes cluster.
- **Docker** for creating custom pipeline images.
- **Python 3.10.12** or later.

### Step-by-Step Installation

1. Clone this repository:
    ```bash
    git clone <repository-url>
    cd <repository-directory>
    ```

2. Install the necessary Python dependencies:
    ```bash
    pip install seaborn qiskit qiskit-machine-learning qiskit-aer numpy pandas scikit-learn matplotlib
    ```

3. Compile the pipeline and upload it to Kubeflow Pipelines:
    ```bash
    dsl-compile --py QSVM_pipeline.py --output qsvm_pipeline.yaml
    ```

4. Deploy the pipeline using the Kubeflow UI or CLI:
    ```bash
    kubectl apply -f qsvm_pipeline.yaml
    ```

## Usage

Once the pipeline is deployed, you can use it to train a quantum SVM model on your dataset. The pipeline includes steps for:

- **Data Preparation**: Load and preprocess data using `pandas`, `numpy`, and `scikit-learn`.
- **Model Training**: Train a QSVM model using Qiskit's quantum computing framework.
- **Model Evaluation**: Evaluate the performance of the model using confusion matrix and accuracy metrics.

### Example

You can run the pipeline with a sample dataset and visualize the model performance metrics, such as accuracy, using the Kubeflow Pipelines UI.

## Components

### 1. QSVM Component

The main component of the pipeline is the `QSVM()` function, which handles the following:

- Data preprocessing (e.g., normalization, train-test split).
- Training a Quantum Support Vector Machine (QSVM) using Qiskit.
- Evaluating model performance.

### Key Libraries:

- **Qiskit**: For quantum circuit simulation and quantum machine learning.
- **scikit-learn**: For data preprocessing and performance evaluation.
- **pandas & numpy**: For data handling and manipulation.
- **matplotlib & seaborn**: For visualization (if required).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
