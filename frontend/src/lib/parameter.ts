export interface SweepValue {
  from: number;
  to: number;
  step: number;
}

export interface Parameter {
  name: string;
  value: string | number | SweepValue;
}
