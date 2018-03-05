import { ParameterDescription } from './parameter';

export interface PipelinePackage {
  createAt?: string;
  id: number;
  description: string;
  name: string;
  parameters: ParameterDescription[];
}
