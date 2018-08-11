import { ListJobsRequest } from './list_jobs_request';

export class ListRunsRequest extends ListJobsRequest {
  public jobId: string;

  constructor(jobId: string, pageSize: number) {
    super(pageSize);
    this.jobId = jobId;
  }

  public toQueryParams(): string {
    return super.toQueryParams() + '&jobId=' + this.jobId.toString();
  }
}

// Valid sortKeys as specified by the backend.
// Note that '' and 'created_at' are considered equivalent.
export enum RunSortKeys {
  CREATED_AT = 'created_at',
  NAME = 'name'
}
