import { ParameterDescription } from './parameter';

export interface PipelinePackage {
  author: string;
  description: string;
  id: string;
  location: string;
  name: string;
  parameters: ParameterDescription[];
  sharedWith?: string;
  tags?: string[];
}
