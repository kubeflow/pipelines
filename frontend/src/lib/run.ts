export interface RunStep {
  name: string;
  start: number;
  end: number;
  state: string;
  outputs: string;
}

export interface Run {
  end: Date;
  id: number;
  instanceId: number;
  parameterValues: { [key: string]: number | string };
  progress: number;
  start: Date;
  state: 'not started' | 'running' | 'errored' | 'completed';
  steps: RunStep[];
}
