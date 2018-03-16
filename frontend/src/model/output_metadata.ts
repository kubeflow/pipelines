export enum PlotType {
  confusion_matrix = 'confusion_matrix',
  roc = 'roc',
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
  plot: PlotMetadata[];
}
