export type PlotType = 'confusion_matrix' | 'roc';

export interface PlotMetadata {
  format: 'csv';
  labels: string[];
  predicted_col: string;
  schema: Array<{ type: string, name: string }>;
  source: string;
  storage: 'gcs';
  target_col: string;
  type: PlotType;
}

export interface OutputMetadata {
  plot: PlotMetadata[];
}
