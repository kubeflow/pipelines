export enum PlotType {
  CONFUSION_MATRIX = 'confusion_matrix',
  ROC = 'roc',
  TENSORBOARD = 'tensorboard',
}

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
  outputs: PlotMetadata[];
}
