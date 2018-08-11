import { ListPipelinesRequest } from './list_pipelines_request';

export class ListJobsRequest extends ListPipelinesRequest {
  public filterBy: string;

  constructor(pageSize: number) {
    super(pageSize);
    this.filterBy = '';
  }

  public toQueryParams(): string {
    // TODO: this isn't actually supported by the backend yet (and won't be for a while.) (5/23)
    return super.toQueryParams() +
        (this.filterBy ? '&filterBy=' + encodeURIComponent(this.filterBy) : '');
  }
}

// Valid sortKeys as specified by the backend.
export enum JobSortKeys {
  CREATED_AT = 'created_at',
  ID = 'id',
  NAME = 'name',
  PIPELINE_ID = 'pipeline_id'
}
