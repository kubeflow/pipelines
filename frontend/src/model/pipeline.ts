import { Parameter } from './parameter';

export interface Pipeline {
  createdAt?: string;
  id?: number;
  author: string;
  description: string;
  name: string;
  packageId: number;
  parameters: Parameter[];
  recurring: boolean;
  recurringIntervalHours: number;
}
