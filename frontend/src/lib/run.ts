import { Parameter } from "src/lib/parameter";

export default interface Run {
  end: Date;
  id: number;
  instanceId: number;
  parameterValues: Parameter[];
  progress: number;
  start: Date;
  state: 'not started' | 'running' | 'errored' | 'completed';
}
