import { ListPipelinesRequest } from './list_pipelines_request';

export class ListJobsRequest extends ListPipelinesRequest {
  public pipelineId: number;

  constructor(pipelineId: number, pageSize: number) {
    super(pageSize);
    this.pipelineId = pipelineId;
  }

  toQueryParams(): string {
    return super.toQueryParams() + '&pipelineId=' + this.pipelineId.toString();
  }
}

// Valid sortKeys as specified by the backend.
// Note that '' and 'created_at' are considered equivalent.
export enum JobSortKeys {
  CREATED_AT = 'created_at',
  NAME = 'name'
}
