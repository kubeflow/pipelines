# Model evaluation with ranking metrics

This tutorial demonstrates how to evaluate ranking or recommendation-style models
in Kubeflow Pipelines using common ranking metrics such as DCG and NDCG.

## What this tutorial shows

- How to define an evaluation component in a Kubeflow Pipeline
- How to compute DCG and NDCG for ranked predictions
- How to interpret ranking metrics in an end-to-end pipeline run

## Structure

- `evaluate_ranking.py`: Defines a simple pipeline with an evaluation step
- `metrics.py`: Implements DCG and NDCG computation on a toy dataset

## Prerequisites

- Kubeflow Pipelines SDK installed
- Basic familiarity with Python-based KFP components

## Notes

This tutorial uses a small, synthetic dataset to keep the example minimal
and focused on evaluation logic rather than model training.
