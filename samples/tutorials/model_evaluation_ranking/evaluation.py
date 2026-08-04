from typing import List, Dict
from .metrics import dcg, ndcg


def evaluate_ranking(
    y_true: List[List[float]],
    y_pred: List[List[int]],
) -> Dict[str, float]:
    """
    Evaluate ranking predictions using DCG and NDCG.

    Args:
        y_true: List of relevance scores per query.
        y_pred: List of ranked indices per query.

    Returns:
        Dictionary with average DCG and NDCG scores.
    """
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


if __name__ == "__main__":
    # Toy example
    y_true = [
        [3, 2, 3, 0, 1, 2],
        [2, 1, 2, 0, 0, 1],
    ]
    y_pred = [
        [0, 2, 5, 1, 4, 3],
        [2, 0, 5, 1, 3, 4],
    ]

    print(evaluate_ranking(y_true, y_pred))
