import { ParameterValue } from './parameter';

export interface JobStep {
  name: string;
  start: number;
  end: number;
  state: string;
  outputs: string;
}

export type JobStatus = 'Not started' | 'Running' | 'Errored' | 'Succeeded';

export type JobOutputPaths = Array<{[k: string]: string}>;

export interface Job {
  createdAt: string;
  finishedAt: string;
  name: string;
  parameterValues: { [key: string]: ParameterValue };
  pipelineId: number;
  progress: number;
  recurring: boolean;
  recurringIntervalHours: number;
  startedAt: string;
  status: JobStatus;
  steps: JobStep[];
}
