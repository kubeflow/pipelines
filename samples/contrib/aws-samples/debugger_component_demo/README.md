# Simple pipeline with train component and debugger

An example pipeline with only [train component](https://github.com/kubeflow/pipelines/tree/master/components/aws/sagemaker/train). The training component has


## Prerequisites

This pipeline uses the exact same setup as [simple_training_pipeline](https://github.com/kubeflow/pipelines/tree/master/samples/contrib/aws-samples/simple_train_pipeline).

## Steps
1. Compile the pipeline:
   `dsl-compile --py debugger-component-demo.py --output debugger-component-demo.tar.gz`
2. In the Kubeflow UI, upload this compiled pipeline specification (the .tar.gz file) and click on create run.
3. Once the pipeline completes, you can view the results of each debugger rule under 'Logs'.

Inputs format to `debug_hook_config` and `debug_rule_config` :
```buildoutcfg
debug_hook_config = {
    "S3OutputPath": "s3://<your_bucket_name>/path/for/data/emission/",
    "LocalPath": "/local/path/for/data/emission/",
    "CollectionConfigurations": [
        {
          "CollectionName": "losses",
          "CollectionParameters": {
            "start_step": "25",
            "end_step": "150"
          }
        }, {
            "CollectionName": "gradient",
            "CollectionParameters": {
                "start_step": "5",
                "end_step": "100"
            }
        }
    ],
    "HookParameters": {
        "save_interval": "10"
    }
}

debug_rule_config = {
    "RuleConfigurationName": "rule_name"
    "RuleEvaluatorImage": "503895931360.dkr.ecr.ap-south-1.amazonaws.com/sagemaker-debugger-rules:latest"
    "RuleParameters": {
        "rule_to_invoke": "LossNotDecreasing",
        "tensor_regex": ".*"
    }
}
```

# Resources
* [Amazon SageMaker Debugger](https://docs.aws.amazon.com/sagemaker/latest/dg/train-debugger.html)
* [Available Frameworks to Use Debugger](https://docs.aws.amazon.com/sagemaker/latest/dg/train-debugger.html#debugger-supported-aws-containers)
* [Debugger Built-In Rules](https://docs.aws.amazon.com/sagemaker/latest/dg/debugger-built-in-rules.html)
* [Debugger Custom Rules](https://docs.aws.amazon.com/sagemaker/latest/dg/debugger-custom-rules.html)
* [Debugger API Examples](https://docs.aws.amazon.com/sagemaker/latest/dg/debugger-createtrainingjob-api.html)

