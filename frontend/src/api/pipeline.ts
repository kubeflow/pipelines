import { Parameter } from './parameter';

export interface Pipeline {
  id: string;
  created_at: string;
  name: string;
  description?: string;
  parameters: Parameter[];
}

export interface PipelineTemplate {
  template: string;
}
