from kfp import dsl
from kfp.compiler import Compiler

from component import evaluate_ranking_component


@dsl.pipeline(
    name="ranking-evaluation-pipeline",
    description="Evaluate ranking models using DCG and NDCG"
)
def ranking_evaluation_pipeline():
    # Toy example data
    y_true = [
        [3.0, 2.0, 1.0],
        [2.0, 1.0, 0.0],
    ]

    y_pred = [
        [0, 1, 2],
        [1, 0, 2],
    ]

    evaluate_ranking_component(
        y_true=y_true,
        y_pred=y_pred,
    )


if __name__ == "__main__":
    Compiler().compile(
        pipeline_func=ranking_evaluation_pipeline,
        package_path="ranking_evaluation_pipeline.yaml",
    )
