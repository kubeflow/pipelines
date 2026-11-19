# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
"""The same typed functions run with the ordinary Argo executor."""

from kfp import dsl


@dsl.component(base_image='kfp-argo-demo:dev', install_kfp_package=False)
def prepare(dataset: dsl.Output[dsl.Dataset]):
    import json

    with open(dataset.path, 'w') as stream:
        json.dump([1, 2, 3, 4], stream)
    dataset.metadata['rows'] = 4


@dsl.component(base_image='kfp-argo-demo:dev', install_kfp_package=False)
def train(
    dataset: dsl.Input[dsl.Dataset], factor: float, model: dsl.Output[dsl.Model]
):
    import json

    with open(dataset.path) as stream:
        numbers = json.load(stream)
    with open(model.path, 'w') as stream:
        json.dump({'prediction': sum(numbers) / len(numbers) * factor}, stream)
    model.metadata['framework'] = 'python'
    model.metadata['dataset_uri'] = dataset.uri


@dsl.component(base_image='kfp-argo-demo:dev', install_kfp_package=False)
def evaluate(
    model: dsl.Input[dsl.Model], metrics: dsl.Output[dsl.Metrics]
) -> float:
    import json

    with open(model.path) as stream:
        prediction = json.load(stream)['prediction']
    score = 1 / (1 + abs(prediction - 3))
    metrics.log_metric('score', score)
    return score


@dsl.pipeline(name='argo-ml-demo')
def pipeline(factor: float = 1.0):
    data = prepare()
    model = train(dataset=data.outputs['dataset'], factor=factor)
    evaluate(model=model.outputs['model'])
