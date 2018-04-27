import { Parameter } from './parameter';

export class Pipeline {
  public id: number;
  public createdAt: string;
  public name: string;
  public description?: string;
  public packageId: number;
  public schedule: string;
  public enabled: boolean;
  public enabledAt: number;
  public parameters: Parameter[];

  constructor() {
    this.createdAt = '';
    this.name = '';
    this.description = '';
    this.packageId = -1;
    this.schedule = '';
    this.enabled = false;
    this.enabledAt = -1;
    this.parameters = [];
  }
}
