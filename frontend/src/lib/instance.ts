import { Parameter } from './parameter';

export interface Instance {
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
