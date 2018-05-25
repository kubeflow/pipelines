import { Parameter } from './parameter';

export interface PipelinePackage {
  id: number;
  // createdAt is SECONDS since epoch
  createdAt: number;
  name: string;
  description?: string;
  parameters: Parameter[];
}
