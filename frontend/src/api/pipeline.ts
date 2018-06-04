import { Parameter } from './parameter';

export class Pipeline {
  public id: number;
  public created_at: string;
  public name: string;
  public description?: string;
  public package_id: number;
  public schedule: string;
  public enabled: boolean;
  public enabled_at: string;
  public parameters?: Parameter[];

  constructor() {
    this.created_at = new Date().toISOString();
    this.name = '';
    this.description = '';
    this.package_id = -1;
    this.schedule = '';
    this.enabled = false;
    this.enabled_at = new Date().toISOString();
    this.parameters = [];
  }
}
