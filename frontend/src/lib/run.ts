export default interface Run {
  end: Date;
  id: number;
  instanceId: number;
  parameterValues: { [key: string]: number | string };
  progress: number;
  start: Date;
  state: 'not started' | 'running' | 'errored' | 'completed';
}
