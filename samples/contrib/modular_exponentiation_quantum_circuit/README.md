# Modular Exponentiation Pipeline

This repository contains a Kubeflow pipeline for executing modular exponentiation using Qiskit. The pipeline is implemented using the `kfp` library and can be compiled into a YAML file for deployment on a Kubeflow Pipelines instance.

## Requirements

To run this pipeline, ensure you have the following installed:

- Python 3.11.7
- Kubeflow Pipelines SDK (`kfp`)

## Pipeline Components
modular_exponentiation Component
This component performs modular exponentiation for a given power number using Qiskit. It returns the quantum circuit diagrams for controlled-U operations for specific values of `a` mod 15.

## modular_exponentiation_pipeline Pipeline
The pipeline defines a single task, modular_exponentiation_task, which executes the modular_exponentiation component with an integer input num.
