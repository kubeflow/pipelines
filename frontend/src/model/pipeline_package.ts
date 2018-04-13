import { Parameter } from './parameter';

export interface PipelinePackage {
  id: number;
  createdAt: string;
  name: string;
  description?: string;
  parameters: Parameter[];
}
