import { ParameterDescription } from './parameter';

export interface PipelinePackage {
  author: string;
  description: string;
  id: number;
  location: string;
  name: string;
  parameters: ParameterDescription[];
  sharedWith: string;
  tags: string[];
}
