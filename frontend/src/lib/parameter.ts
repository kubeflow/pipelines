export type ParameterValue = string | number;

export interface Parameter {
  name: string;
  isSweep: boolean;
  value: ParameterValue;
  from: number;
  to: number;
  step: number;
}
