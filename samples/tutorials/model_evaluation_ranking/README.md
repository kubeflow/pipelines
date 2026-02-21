# Model evaluation with ranking metrics

This sample demonstrates how ranking or recommendation-style models can be
evaluated in Kubeflow Pipelines using common ranking metrics such as
Discounted Cumulative Gain (DCG) and Normalized DCG (NDCG).

The goal of this example is to show how evaluation logic can be structured
and integrated into a Kubeflow Pipeline, rather than to provide a full
end-to-end training workflow.

## What this sample demonstrates

- How to implement DCG and NDCG metrics for ranked predictions
- How to encapsulate ranking evaluation logic in a reusable evaluation helper
- How to wrap the evaluation logic as a Kubeflow Pipelines component
- How an evaluation component can be wired into a simple Kubeflow Pipeline

## File structure

- `metrics.py`  
  Implements DCG and NDCG computations.

- `evaluation.py`  
  Provides a helper function to evaluate ranking predictions across multiple
  queries and aggregate metrics.

- `component.py`  
  Defines a Kubeflow Pipelines component that performs ranking evaluation.

- `pipeline.py`  
  Illustrates how the evaluation component can be invoked inside a Kubeflow
  Pipeline using a small, synthetic example.

## Usage

This sample is intended as a **reference example**.

The pipeline definition in `pipeline.py` is provided to demonstrate how the
ranking evaluation component can be integrated into a Kubeflow Pipeline.
It is not intended to be executed as-is by end users.

Users can adapt the evaluation component and pipeline structure shown here
to fit their own Kubeflow Pipelines workflows and datasets.

## Notes

- The example uses a small synthetic dataset to keep the focus on evaluation
  logic rather than model training or infrastructure configuration.
- No changes to Kubeflow Pipelines core code are required to use this sample.
