import math
from typing import List


def dcg(relevances: List[float]) -> float:
    """
    Compute Discounted Cumulative Gain (DCG).

    Args:
        relevances: Relevance scores in ranked order.

    Returns:
        DCG value.
    """
    return sum(
        rel / math.log2(idx + 2)
        for idx, rel in enumerate(relevances)
    )


def ndcg(relevances: List[float]) -> float:
    """
    Compute Normalized Discounted Cumulative Gain (NDCG).

    Args:
        relevances: Relevance scores in ranked order.

    Returns:
        NDCG value between 0 and 1.
    """
    ideal = sorted(relevances, reverse=True)
    ideal_dcg = dcg(ideal)

    if ideal_dcg == 0:
        return 0.0

    return dcg(relevances) / ideal_dcg


if __name__ == "__main__":
    # Example usage with a toy ranking
    example_relevances = [3, 2, 3, 0, 1, 2]
    print("DCG:", dcg(example_relevances))
    print("NDCG:", ndcg(example_relevances))
