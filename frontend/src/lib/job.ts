import { ParameterValue } from '../lib/parameter';

export interface JobStep {
  name: string;
  start: number;
  end: number;
  state: string;
  outputs: string;
}

export interface Job {
  end: number;
  id: number;
  pipelineId: number;
  parameterValues: { [key: string]: ParameterValue };
  progress: number;
  recurring: boolean;
  recurringIntervalHours: number;
  start: number;
  state: 'not started' | 'running' | 'errored' | 'completed';
  steps: JobStep[];
}
