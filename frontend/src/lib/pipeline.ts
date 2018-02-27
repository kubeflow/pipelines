import { Parameter } from './parameter';

export interface Pipeline {
  author: string;
  description: string;
  ends: number;
  id?: number;
  name: string;
  packageId: number;
  parameterValues: Parameter[];
  recurring: boolean;
  recurringIntervalHours: number;
  starts: number;
  tags: string[];
}
