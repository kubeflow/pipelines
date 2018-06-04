import { Parameter } from './parameter';

export interface PipelinePackage {
  id: number;
  created_at: string;
  name: string;
  description?: string;
  parameters: Parameter[];
}
