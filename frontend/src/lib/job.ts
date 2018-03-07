import { ParameterValue } from '../lib/parameter';

export interface JobStep {
  name: string;
  start: number;
  end: number;
  state: string;
  outputs: string;
}

export type JobStatus = 'Not started' | 'Running' | 'Errored' | 'Succeeded';

export interface Job {
  end: number;
  id: number;
  pipelineId: number;
  parameterValues: { [key: string]: ParameterValue };
  progress: number;
  recurring: boolean;
  recurringIntervalHours: number;
  start: number;
  status: JobStatus;
  steps: JobStep[];
}
