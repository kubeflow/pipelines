from .legacy_data_passing import data_passing_pipeline
from kfp.samples.test.utils import run_pipeline_func, TestCase
from kfp.dsl import PipelineExecutionMode

run_pipeline_func([
    TestCase(
        pipeline_func=data_passing_pipeline,
        mode=PipelineExecutionMode.V1_LEGACY,
    )
])
