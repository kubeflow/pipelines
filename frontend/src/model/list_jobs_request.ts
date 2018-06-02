export class ListJobsRequest {
  public filterBy: string;
  public orderAscending: boolean;
  public pageSize: number;
  public pageToken: string;
  public pipelineId: number;
  public sortBy: string;

  constructor(pipelineId: number, pageSize: number) {
    this.filterBy = '';
    this.pageSize = pageSize;
    this.pageToken = '';
    this.pipelineId = pipelineId;
    this.sortBy = '';
    this.orderAscending = true;
  }
}

// Valid sortKeys as specified by the backend.
// Note that '' and 'created_at' are considered equivalent.
export enum JobSortKeys {
  CREATED_AT = 'created_at',
  NAME = 'name'
}
