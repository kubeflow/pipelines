import { ParameterValue } from '../lib/parameter';

export interface RunStep {
  name: string;
  start: number;
  end: number;
  state: string;
  outputs: string;
}

export interface Run {
  end: number;
  id: number;
  instanceId: number;
  parameterValues: { [key: string]: ParameterValue };
  progress: number;
  recurring: boolean;
  recurringIntervalHours: number;
  start: number;
  state: 'not started' | 'running' | 'errored' | 'completed';
  steps: RunStep[];
}
