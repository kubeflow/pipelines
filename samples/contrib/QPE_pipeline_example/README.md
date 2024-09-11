# Quantum Phase Estimation (QPE) Pipeline

This repository contains a Kubeflow pipeline for running a Quantum Phase Estimation (QPE) algorithm using Qiskit, a quantum computing framework. The pipeline leverages the kfp library to define and compile the pipeline.

## Requirements
To run this pipeline, you need to have the following installed:

Python 3.11.7 (base-image)
Kubeflow Pipelines SDK (kfp)

In pipeline:

`Qiskit Aer`

`Qiskit`

`pylatexenc`

`ipywidgets`

`matplotlib`

## Pipeline Components
qpe Component

This component runs the Quantum Phase Estimation algorithm. It takes an integer `n` as an input, which specifies the number of counting qubits to use in the algorithm.
