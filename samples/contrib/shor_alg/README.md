# Shor's Algorithm Pipeline

This repository contains a Kubeflow pipeline for executing Shor's Algorithm, a quantum algorithm used for integer factorization. The pipeline is implemented using the `kfp` library and can be compiled into a YAML file for deployment on a Kubeflow Pipelines instance.

## Requirements

To run this pipeline, ensure you have the following installed:

- Kubeflow Pipelines SDK (`kfp`)

## Pipeline Components

shor_alg Component
This component implements Shor's Algorithm to factorize an integer num. It returns the factors p and q of num as outputs.

## shor_alg_pipeline Pipeline

The pipeline defines a single task, shor_alg_task, which executes the shor_alg component with an integer input num.
