import { Parameter } from './parameter';

export class Pipeline {
  public createdAt: string;
  public id?: number;
  public author: string;
  public description: string;
  public name: string;
  public packageId: number;
  public parameters: Parameter[];
  public recurring: boolean;
  public recurringIntervalHours: number;

  constructor() {
    this.createdAt = '';
    this.author = '';
    this.description = '';
    this.name = '';
    this.packageId = -1;
    this.parameters = [];
    this.recurring = false;
    this.recurringIntervalHours = -1;
  }
}
