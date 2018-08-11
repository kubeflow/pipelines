import { Parameter } from './parameter';

export interface Pipeline {
  id: number;
  created_at: string;
  name: string;
  description?: string;
  parameters: Parameter[];
}

export interface PipelineTemplate {
  template: string;
}
