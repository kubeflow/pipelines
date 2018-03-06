import { Parameter } from './parameter';

export interface Pipeline {
  createAt?: string;
  id?: number;
  author: string;
  description: string;
  ends: number;
  name: string;
  packageId: number;
  parameters: Parameter[];
  recurring: boolean;
  recurringIntervalHours: number;
  starts: number;
}
