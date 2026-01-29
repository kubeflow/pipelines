from typing import List, Dict
from kfp.dsl import component


@component(
    base_image="python:3.10",
    packages_to_install=[]
)
def evaluate_ranking_component(
    y_true: List[List[float]],
    y_pred: List[List[int]],
) -> Dict[str, float]:
    """
    Kubeflow component to evaluate ranking predictions using DCG and NDCG.
    """
    from metrics import dcg, ndcg

    dcg_scores = []
    ndcg_scores = []

    for true_relevances, ranking in zip(y_true, y_pred):
        ranked_relevances = [true_relevances[i] for i in ranking]
        dcg_scores.append(dcg(ranked_relevances))
        ndcg_scores.append(ndcg(ranked_relevances))

    return {
        "mean_dcg": sum(dcg_scores) / len(dcg_scores),
        "mean_ndcg": sum(ndcg_scores) / len(ndcg_scores),
    }
