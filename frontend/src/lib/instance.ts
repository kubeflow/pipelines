import { Parameter } from "src/lib/parameter";

export interface Instance {
  author: string;
  description: string;
  ends: Date;
  id: number;
  name: string;
  parameterValues: Parameter[];
  recurring: boolean;
  recurringIntervalHours: number;
  starts: Date;
  tags: string[];
  templateId: number;
}
