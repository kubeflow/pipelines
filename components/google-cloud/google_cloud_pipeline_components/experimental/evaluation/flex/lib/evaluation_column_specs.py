"""Convenience wrapper around prediction component specification."""

import dataclasses
from typing import Optional

from lib import column_spec

ColumnSpec = column_spec.ColumnSpec


@dataclasses.dataclass(frozen=True)
class EvaluationColumnSpecs():
  """Class encapsulating column specs necessary for model evaluation.

  Attributes:
    ground_truth_column_spec: Spec for the column containing ground truth.
    example_weight_column_spec: Spec for the column containing example weights,
      if any.
    predicted_score_column_spec: Spec for the column containing scores returned
      by the model.
    predicted_label_column_spec: Spec for the column containining names for the
      classes being scored by the model, if model returns them.
    predicted_label_id_column_spec: Spec for the column containining ids for the
      classes being scored by the model, if model returns them.
  """

  ground_truth_column_spec: ColumnSpec
  example_weight_column_spec: Optional[ColumnSpec] = None
  predicted_score_column_spec: Optional[ColumnSpec] = None
  predicted_label_column_spec: Optional[ColumnSpec] = None
  predicted_label_id_column_spec: Optional[ColumnSpec] = None

  def __post_init__(self):
    self._validate_()

  def _validate_(self):
    """Validates that inputs are correct."""

    if not self.predicted_score_column_spec:
      raise ValueError('predicted_score_column_spec is not provided')
