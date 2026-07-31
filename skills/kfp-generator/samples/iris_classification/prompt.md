@kfp-generator Use our registered agent skill to create a complete Kubeflow Pipeline sample named 'samples/core/generated_agent_pipelines/iris_training.py'. 

The pipeline must:
1. Download the Iris dataset using sklearn.
2. Perform a basic ETL step (preprocess features or split into train/test sets).
3. Train a simple model.
4. Evaluate the model metrics.

Make sure to include a compilation block at the bottom of the script. Run our local evaluation script with --run --host http://localhost:8080 on it to guarantee a valid KFP v2 YAML specification and verify live backend execution.