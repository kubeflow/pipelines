"""Math functions for evaluation metrics."""

from typing import List
EPS = 1.0e-7


def prob_fix(x: float) -> float:
  """Maps x==1. -> 1.-EPS and x==0. -> EPISLON."""
  if isinstance(x, float):
    if x < 0.:
      raise ValueError('Negative numbers invalid.')
    return x - EPS if x == 1. else max(EPS, x)
  raise TypeError(type(x), 'Cannot process value of this type.')


def prob_fix_list(x: List[float]) -> List[float]:
  """Call prob_fix on a list of floats."""
  return [prob_fix(v) for v in x]
