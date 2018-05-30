export class ListJobsRequest {
  public filterBy: string;
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
  }
}
