export interface MetadataPlot {
  type: 'roc';
  format: 'csv';
  source: string;
  schema: string;
  predicted_col: string;
  target_col: string;
}

export interface OutputMetadata {
  plot: MetadataPlot[];
}
