export type ParameterValue = string | number;

export interface Parameter {
  name: string;
  description?: string;
  value?: ParameterValue;
}
