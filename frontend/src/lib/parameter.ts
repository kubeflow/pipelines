export type ParameterValue = string | number;

export interface ParameterDescription {
  name: string;
  description: string;
}

export interface Parameter {
  name: string;
  value: ParameterValue;
}
