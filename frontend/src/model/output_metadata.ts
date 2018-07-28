export enum PlotType {
  CONFUSION_MATRIX = 'confusion_matrix',
  ROC = 'roc',
  TABLE = 'table',
  TENSORBOARD = 'tensorboard',
  WEB_APP = 'web-app',
}

export interface PlotMetadata {
  format?: 'csv';
  header?: string[];
  labels?: string[];
  predicted_col?: string;
  schema?: Array<{ type: string, name: string }>;
  source: string;
  storage?: 'gcs';
  target_col?: string;
  type: PlotType;
}

export interface OutputMetadata {
  outputs: PlotMetadata[];
}
