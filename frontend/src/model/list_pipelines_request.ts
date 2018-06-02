export class ListPipelinesRequest {
  public filterBy: string;
  public orderAscending: boolean;
  public pageSize: number;
  public pageToken: string;
  public sortBy: string;

  constructor(pageSize: number) {
    this.filterBy = '';
    this.pageSize = pageSize;
    this.pageToken = '';
    this.sortBy = '';
    this.orderAscending = true;
  }
}

// Valid sortKeys as specified by the backend.
export enum PipelineSortKeys {
  CREATED_AT = 'created_at',
  ID = 'id',
  NAME = 'name',
  PACKAGE_ID = 'package_id'
}
