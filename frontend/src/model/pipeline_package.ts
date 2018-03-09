import { ParameterDescription } from './parameter';

export interface PipelinePackage {
  createdAt?: string;
  id: number;
  description: string;
  name: string;
  parameters: ParameterDescription[];
}
