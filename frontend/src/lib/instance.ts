export interface Parameter {
  name: string;
  value: string | number;
}

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
